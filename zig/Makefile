.PHONY: build clean format format-check

build:
	zig build

clean:
	rm -rf zig-out zig-cache .zig-cache

format:
	zig fmt src/

format-check:
	zig fmt --check src/
