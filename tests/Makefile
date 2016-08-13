# .../src/github.com/strickyak/prego/tests/Makefile

all: Run-z2
	echo Tests OKAY

clean:
	-rm -f tmp_*

Run-z2:
	go run ../main.go < z2.po > tmp_z2_.go
	go run tmp_z2_.go > tmp_z2_.out
	diff z2.expect tmp_z2_.out
