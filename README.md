## govanish

It might not be a surprise to you that your code can simply disappear from the compiled binary for multiple reasons.

Sometimes, this is beneficial (for example, if you define a constant boolean for debugging purposes which is explicitly set to false in the code).

However, sometimes this can actually be a sign of a subtle bug. For instance, consider the following snippet:

```go
type E struct{ Desc string }

func (e *E) Error() string { return e.Desc }
func api() *E              { return nil }

func use() {
    var err error = api()
    if err == nil {
       panic("panic all the time!")
    }
}
```

This code will never panic because the condition err == nil is always false (see https://trstringer.com/go-nil-interface-and-interface-with-nil-concrete-value/ for a detailed explanation).

The thing is, there are multiple subtle nuances in Go which can lead the compiler to remove your code unexpectedly.

`govanish` linter helps to identify such instances in your code. To test it, you can run this linter against its own source code:
```
$> go run main.go
2023/12/20 03:07:51 cwd: /home/sivukhin/code/govanish
2023/12/20 03:07:51 ready to compile project for assembly inspection
2023/12/20 03:07:51 assembly information were indexed, prepare it for queries
2023/12/20 03:07:51 assembly information were prepared, ready to process AST
2023/12/20 03:07:51 it seems like your code vanished from compiled binary: func=[use], file=[/home/sivukhin/code/govanish/main.go], line=[25]
```