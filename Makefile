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

TEST_BIN := $(OBJ)/test_rpiInfo
TEST_CC  := gcc

.PHONY: test
test: $(TEST_BIN)
	./$(TEST_BIN)

$(TEST_BIN): test/test_rpiInfo.c hardware/rpiInfo/rpiInfo.c hardware/rpiInfo/rpiInfo.h
	$(TEST_CC) $(CFLAGS) -I hardware/rpiInfo -I project/ -o $@ test/test_rpiInfo.c hardware/rpiInfo/rpiInfo.c

FMT_SRCS = find $(SRCDIRS) test/ -type f \( -name '*.c' -o -name '*.h' \) -print0

format:
	$(FMT_SRCS) | xargs -0 clang-format -i

format-check:
	$(FMT_SRCS) | xargs -0 clang-format --dry-run --Werror
