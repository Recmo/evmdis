
all:
	go build -o evmdis.exe ./evmdis

test: all
	solc test/contract.sol --bin --asm -o test
	./evmdis.exe < test/Test.bin
