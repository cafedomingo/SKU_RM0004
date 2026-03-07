.PHONY: build clean format format-check clippy

build:
	cargo build --release

clean:
	cargo clean

format:
	cargo fmt

format-check:
	cargo fmt -- --check

clippy:
	cargo clippy -- -D warnings
