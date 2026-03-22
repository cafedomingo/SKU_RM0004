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

TEST_CC := gcc
TESTS   := test_rpiInfo test_format test_theme test_st7735_fb

test_rpiInfo_SRCS   := test/test_rpiInfo.c hardware/rpiInfo/rpiInfo.c
test_format_SRCS    := test/test_format.c project/format.c
test_theme_SRCS     := test/test_theme.c project/theme.c
test_st7735_fb_SRCS := test/test_st7735_fb.c hardware/st7735/st7735_fb.c hardware/st7735/fonts.c

.PHONY: test
test: $(TESTS:%=$(OBJ)/%)
	@for t in $^; do ./$$t || exit 1; done

define TEST_RULE
$(OBJ)/$(1): $$($(1)_SRCS)
	$$(TEST_CC) $$(CFLAGS) $$(INCLUDE) -o $$@ $$^
endef
$(foreach t,$(TESTS),$(eval $(call TEST_RULE,$(t))))

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
