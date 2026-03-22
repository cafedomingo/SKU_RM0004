#include "dashboard.h"
#include "diagnostic.h"
#include "sparkline.h"
#include "st7735.h"
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <zlib.h>

extern uint8_t display_fb[];

#define DEFAULT_SCALE 5

/* ── Minimal PNG writer ──────────────────────────────────────────── */

static uint32_t crc_table[256];
static int crc_table_ready;

static void build_crc_table(void) {
    for (uint32_t n = 0; n < 256; n++) {
        uint32_t c = n;
        for (int k = 0; k < 8; k++)
            c = (c & 1) ? 0xEDB88320 ^ (c >> 1) : c >> 1;
        crc_table[n] = c;
    }
    crc_table_ready = 1;
}

static uint32_t update_crc(uint32_t crc, const uint8_t *buf, uint32_t len) {
    if (!crc_table_ready) build_crc_table();
    for (uint32_t i = 0; i < len; i++)
        crc = crc_table[(crc ^ buf[i]) & 0xFF] ^ (crc >> 8);
    return crc;
}

static void put32be(uint8_t *p, uint32_t v) {
    p[0] = (v >> 24) & 0xFF;
    p[1] = (v >> 16) & 0xFF;
    p[2] = (v >> 8) & 0xFF;
    p[3] = v & 0xFF;
}

static void write_chunk(FILE *f, const char *type, const uint8_t *data, uint32_t len) {
    uint8_t hdr[8];
    put32be(hdr, len);
    memcpy(hdr + 4, type, 4);
    fwrite(hdr, 1, 8, f);
    if (len > 0) fwrite(data, 1, len, f);
    uint32_t crc = update_crc(0xFFFFFFFF, hdr + 4, 4);
    if (len > 0) crc = update_crc(crc, data, len);
    crc ^= 0xFFFFFFFF;
    uint8_t crc_buf[4];
    put32be(crc_buf, crc);
    fwrite(crc_buf, 1, 4, f);
}

static void write_png(const char *filename, int scale) {
    uint32_t w = ST7735_WIDTH * scale;
    uint32_t h = ST7735_HEIGHT * scale;

    FILE *f = fopen(filename, "wb");
    if (!f) {
        perror(filename);
        return;
    }

    /* PNG signature */
    static const uint8_t sig[] = {0x89, 'P', 'N', 'G', '\r', '\n', 0x1A, '\n'};
    fwrite(sig, 1, 8, f);

    /* IHDR */
    uint8_t ihdr[13];
    put32be(ihdr, w);
    put32be(ihdr + 4, h);
    ihdr[8] = 8;  /* bit depth */
    ihdr[9] = 2;  /* color type: RGB */
    ihdr[10] = 0; /* compression */
    ihdr[11] = 0; /* filter */
    ihdr[12] = 0; /* interlace */
    write_chunk(f, "IHDR", ihdr, 13);

    /* Build raw image data: filter_byte(0) + RGB for each row */
    uint32_t raw_row = 1 + w * 3;
    uint32_t raw_size = raw_row * h;

    uint8_t *raw = malloc(raw_size);
    if (!raw) {
        perror("malloc");
        fclose(f);
        return;
    }

    /* Convert RGB565 framebuffer to scaled RGB888 */
    uint8_t src_row[ST7735_WIDTH * 3];
    for (uint32_t sy = 0; sy < (uint32_t)ST7735_HEIGHT; sy++) {
        for (uint32_t sx = 0; sx < (uint32_t)ST7735_WIDTH; sx++) {
            uint32_t off = (sy * ST7735_WIDTH + sx) * 2;
            uint16_t c = (uint16_t)(display_fb[off] << 8) | display_fb[off + 1];
            uint8_t r5 = (c >> 11) & 0x1F;
            uint8_t g6 = (c >> 5) & 0x3F;
            uint8_t b5 = c & 0x1F;
            src_row[sx * 3] = (r5 << 3) | (r5 >> 2);
            src_row[sx * 3 + 1] = (g6 << 2) | (g6 >> 4);
            src_row[sx * 3 + 2] = (b5 << 3) | (b5 >> 2);
        }

        for (int dy = 0; dy < scale; dy++) {
            uint32_t dst_y = sy * scale + dy;
            uint8_t *dst = raw + dst_y * raw_row;
            *dst++ = 0; /* filter: none */
            for (uint32_t sx = 0; sx < (uint32_t)ST7735_WIDTH; sx++)
                for (int dx = 0; dx < scale; dx++) {
                    memcpy(dst, src_row + sx * 3, 3);
                    dst += 3;
                }
        }
    }

    /* Compress with zlib */
    uLong comp_bound = compressBound(raw_size);
    uint8_t *comp = malloc(comp_bound);
    if (!comp) {
        perror("malloc");
        free(raw);
        fclose(f);
        return;
    }

    uLong comp_size = comp_bound;
    int zret = compress2(comp, &comp_size, raw, raw_size, 9);
    free(raw);
    if (zret != Z_OK) {
        fprintf(stderr, "compress2 failed: %d\n", zret);
        free(comp);
        fclose(f);
        return;
    }

    write_chunk(f, "IDAT", comp, comp_size);
    free(comp);

    /* IEND */
    write_chunk(f, "IEND", NULL, 0);

    fclose(f);
    printf("  %s (%ux%u)\n", filename, w, h);
}

/* ── Main ────────────────────────────────────────────────────────── */

int main(int argc, char **argv) {
    int scale = DEFAULT_SCALE;
    if (argc > 1) scale = atoi(argv[1]);
    if (scale < 1 || scale > 10) scale = DEFAULT_SCALE;

    printf("Rendering screenshots at %dx scale...\n", scale);

    /* Dashboard */
    memset(display_fb, 0, ST7735_WIDTH * ST7735_HEIGHT * 2);
    lcd_display_dashboard();
    write_png("docs/dashboard.png", scale);

    /* Sparkline */
    memset(display_fb, 0, ST7735_WIDTH * ST7735_HEIGHT * 2);
    sparkline_invalidate();
    sparkline_state_t state = {
        .cpu_history = {22, 35, 28, 45, 52, 38, 61, 73, 55, 42, 68, 50, 47},
        .ram_history = {40, 42, 45, 48, 50, 53, 55, 57, 58, 60, 61, 62, 63},
        .ticker_phase = 0,
    };
    lcd_display_sparkline(&state);
    write_png("docs/sparkline.png", scale);

    /* Diagnostic page 0 */
    memset(display_fb, 0, ST7735_WIDTH * ST7735_HEIGHT * 2);
    diag_refresh_data();
    lcd_display_diagnostic_page(0);
    write_png("docs/diagnostic_p0.png", scale);

    /* Diagnostic page 1 */
    memset(display_fb, 0, ST7735_WIDTH * ST7735_HEIGHT * 2);
    lcd_display_diagnostic_page(1);
    write_png("docs/diagnostic_p1.png", scale);

    printf("Done.\n");
    return 0;
}
