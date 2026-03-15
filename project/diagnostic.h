#ifndef __DIAGNOSTIC_H__
#define __DIAGNOSTIC_H__

#include <stdint.h>

#define DIAG_TOTAL_ROWS 15
#define DIAG_NUM_PAGES  2

void diag_refresh_data(void);
void lcd_display_diagnostic_page(int page);

#endif /* __DIAGNOSTIC_H__ */
