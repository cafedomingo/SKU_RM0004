#ifndef __DIAGNOSTIC_H__
#define __DIAGNOSTIC_H__

#include <stdint.h>

#define DIAG_VISIBLE_ROWS 8
#define DIAG_TOTAL_ROWS   21
#define DIAG_SCROLL_SECS  1

void diag_refresh_data(void);
void lcd_display_diagnostic(uint8_t scroll_offset);

#endif /* __DIAGNOSTIC_H__ */
