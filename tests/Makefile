# .../src/github.com/strickyak/prego/tests/Makefile

all: Run-demo Run-demo-noinline

clean:
	-rm -f tmp_*

Run-demo:
	go run ../main.go < demo.po > tmp_demo_.go
	go run tmp_demo_.go > tmp_demo_.out
	diff demo.expect tmp_demo_.out
	echo Tests OKAY

Run-demo-noinline:
	go run ../main.go < demo.po > tmp_demo_.go --noinline
	go run tmp_demo_.go > tmp_demo_.out
	diff demo.expect tmp_demo_.out
	echo Tests OKAY
