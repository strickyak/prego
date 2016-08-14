# .../src/github.com/strickyak/prego/tests/Makefile

all: Run-demo

# Go does not allow `_` at front of .go files, but it does allow `,`.
# So we mark temporary files with `,`.
clean:
	-rm -f ,_tmp_*

Run-demo:
	go run ../main.go < demo.po > ,_tmp_demo_.go
	go run ,_tmp_demo_.go > ,_tmp_demo_.out
	diff demo.expect ,_tmp_demo_.out
	echo Tests OKAY
