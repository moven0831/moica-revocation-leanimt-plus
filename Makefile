.PHONY: bench-real bench-real-fetch bench-real-clean help

help:
	@echo "Top-level targets (server-specific targets are in server/Makefile):"
	@echo "  bench-real         Run the LeanIMT+ vs SMT real-data benchmark."
	@echo "  bench-real-fetch   Only download CRL DERs to bench/.cache/."
	@echo "  bench-real-clean   Remove bench/.cache/."

bench-real:
	./bench/run.sh

bench-real-fetch:
	./bench/fetch.sh

bench-real-clean:
	rm -rf bench/.cache
