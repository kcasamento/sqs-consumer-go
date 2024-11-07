.PHONY test:
test:
	go test -v -race ./...

.PHONY bench-harvester:
bench-harvester:
	rm -rf .out && \
	mkdir .out && \
	go test \
		-bench=. \
		-count=10 \
		-race \
		-benchmem \
		-memprofile	.out/mem.out \
		-cpuprofile .out/cpu.out \
		./internal/harvester/... | tee .out/stats.txt

.PHONY bench-stats:
bench-stats:
	benchstat ./.out/stats.txt

