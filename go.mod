module github.com/wayan/bocian-go

go 1.18

require (
	github.com/cpuguy83/go-md2man/v2 v2.0.1 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/urfave/cli/v2 v2.4.0
)

replace github.com/wayan/mergeexp => ../mergeexp

require github.com/wayan/mergeexp v0.0.0-20220404171102-e985bcc80bae
