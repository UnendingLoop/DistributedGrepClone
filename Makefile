APP=mygrep
CMD=./cmd/app.go

PORTS=8080 8081 8082 8083
PATTERN=abc
FILE=test.txt

build:
	go build -o $(APP) $(CMD)

run-slaves:
	@for p in $(PORTS); do \
		./$(APP) -mode=slave -addr=$$p & \
	done
	@sleep 2

run-master:
	./$(APP) -mode=master \
		-node=localhost:8080 \
		-node=localhost:8081 \
		-node=localhost:8082 \
		-node=localhost:8083 \
		-F -n $(PATTERN) $(FILE) > my.out

run-grep:
	grep -F -n $(PATTERN) $(FILE) > ref.out

compare:
	diff -w my.out ref.out

clean:
	pkill $(APP) || true
	rm ref.out
	rm my.out

integration: build run-slaves run-master run-grep compare clean
	@echo "Integration test passed"