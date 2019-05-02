.PHONY: all
.PHONY: tests
.SILENT: build
.SILENT: cleanup

all: build
	@echo "Look at the settings in config.yaml. (You can make changes in the config on the fly ✈️ )"
	(go run coordinator.go)

build:
	-(go get)

unit_tests:
	@echo "\n\n--- Running unit tests ---"
	(cd worker; go test -v; cd ..)

sanity_tests: unit_tests
	@echo "\n\n--- Running sanity tests ---"
	(rm -rf db/)
	(go run test/dummy.go)

cleanup:
	-(bash cleanup.sh)

tests: sanity_tests cleanup
	@echo "\n\n------"
	