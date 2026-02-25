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
$(OBJS) : obj/%.o : %.c
	$(CC) $(CFLAGS) -c $(INCLUDE) -o $@ $<


clean:
	rm -rf $(OBJ)
	rm -rf $(TARGET)

format:
	find . -name '*.c' -o -name '*.h' | grep -v fonts | xargs clang-format -i

format-check:
	find . -name '*.c' -o -name '*.h' | grep -v fonts | xargs clang-format --dry-run --Werror
