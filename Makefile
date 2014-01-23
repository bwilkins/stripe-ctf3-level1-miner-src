EXE = gominer

all: $(EXE)

$(EXE):
	go build -o $(EXE) miner.go

clean:
	rm -f $(EXE)

.PHONY: clean all