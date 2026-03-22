TARGET := display
CC     ?= gcc
CFLAGS := -Wall -Wextra -Wno-unused-parameter -O2 -D_FORTIFY_SOURCE=2 -fstack-protector-strong

OBJ := obj

mkfile_path := $(shell pwd)/$(lastword $(MAKEFILE_LIST))
dir=$(shell dirname $(mkfile_path))
$(shell mkdir -p $(dir)/$(OBJ))

SRCDIRS :=  project/ \
			hardware/rpiInfo \
			hardware/st7735

SRCS := $(foreach dir, $(SRCDIRS), $(wildcard $(dir)/*.c))
NOT_DIR :=$(notdir $(SRCS))
OBJS := $(patsubst %.c, $(OBJ)/%.o, $(NOT_DIR))

INCLUDE := $(patsubst %, -I %, $(SRCDIRS))

VPATH := $(SRCDIRS)

$(TARGET):$(OBJS)
	$(CC) $(CFLAGS) -o $@ $^
DEPS := $(OBJS:.o=.d)

$(OBJS) : obj/%.o : %.c
	$(CC) $(CFLAGS) -MMD -MP -c $(INCLUDE) -o $@ $<

-include $(DEPS)


clean:
	rm -rf $(OBJ)
	rm -rf $(TARGET)

TEST_CC  := gcc

TEST_RPIINFO := $(OBJ)/test_rpiInfo
TEST_FORMAT  := $(OBJ)/test_format
TEST_THEME   := $(OBJ)/test_theme

.PHONY: test
test: $(TEST_RPIINFO) $(TEST_FORMAT) $(TEST_THEME)
	./$(TEST_RPIINFO)
	./$(TEST_FORMAT)
	./$(TEST_THEME)

$(TEST_RPIINFO): test/test_rpiInfo.c hardware/rpiInfo/rpiInfo.c hardware/rpiInfo/rpiInfo.h
	$(TEST_CC) $(CFLAGS) -I hardware/rpiInfo -I project/ -o $@ test/test_rpiInfo.c hardware/rpiInfo/rpiInfo.c

$(TEST_FORMAT): test/test_format.c project/format.c project/format.h
	$(TEST_CC) $(CFLAGS) -I project/ -I hardware/rpiInfo -o $@ test/test_format.c project/format.c

$(TEST_THEME): test/test_theme.c project/theme.c project/theme.h
	$(TEST_CC) $(CFLAGS) -I project/ -I hardware/st7735 -o $@ test/test_theme.c project/theme.c

SCREENSHOT := $(OBJ)/screenshot
SCREENSHOT_SRCS := tools/screenshot.c tools/mock_st7735.c tools/mock_rpiInfo.c \
                   project/dashboard.c project/sparkline.c \
                   project/format.c project/theme.c \
                   hardware/st7735/fonts.c hardware/st7735/st7735_fb.c

.PHONY: screenshot
screenshot: $(SCREENSHOT)
	@mkdir -p docs
	./$(SCREENSHOT)

$(SCREENSHOT): $(SCREENSHOT_SRCS)
	$(TEST_CC) $(CFLAGS) $(INCLUDE) -I hardware/st7735 -o $@ $(SCREENSHOT_SRCS) -lz

FMT_SRCS = find $(SRCDIRS) test/ tools/ -type f \( -name '*.c' -o -name '*.h' \) -print0

format:
	$(FMT_SRCS) | xargs -0 clang-format -i

format-check:
	$(FMT_SRCS) | xargs -0 clang-format --dry-run --Werror
