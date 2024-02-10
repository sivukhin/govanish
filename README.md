# govanish

`govanish` will highlight lines of code which most likely were removed by the compiler from the final executable binary.

## Usage

```bash
$> go install github.com/sivukhin/govanish@latest
$> cd /path/to/your/module && govanish                # go to your module and run govanish from root directory with go.mod file
$> govanish -path /path/to/your/module                # or you can provide path to the root directory as first argument
$> govanish -path /path/to/your/module -format github # you can format errors in format for GitHub actions
```

## Purpose

It might not be a surprise to you that your code can simply disappear from the compiled binary for multiple reasons.

Sometimes this is good. For example, if you define some constant `debug = false` for local debugging, Go compiler will remove conditional branches which depends on this variable and program will not waste CPU on useless checks. Also, compiler can optimize some operations and replace sequence of statements with more performance combination of equivalent instructions.   

However, removal of some parts of the code can actually be a sign of a subtle bugs, because usually we don't want for code to suddenly disappear - why did we write it then?. 

For example, consider following snippet:
```go
func handle() error {
    fErr := F()
    if fErr != nil {
        return fErr
    }
    gErr := G()
    if fErr != nil {
        return gErr    
    }
    return nil
}
```

Here, we obviously made a typo and checked `fErr` again instead of checking `gErr`. And in this case compiler removes second error check because it duplicates first one and actually useless! 

Consider more subtle snippet:
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

This code will never panic because the condition `err == nil` is always `false` (see https://trstringer.com/go-nil-interface-and-interface-with-nil-concrete-value/ for a detailed explanation). And in this case compiler also can prove that condition will never be satisfied and remove it from the binary.

The thing is, there are multiple subtle nuances in Go which can lead to the unwanted code removal by the compiler.

`govanish` helps to identify such instances in your code. Just run it from the base directory of your Go module or provide path to the module as a first argument

```shell
$> govanish
$> govanish -path projects/go/cache # projects/go/cache/ is a module root and has go.mod file
2024/01/04 19:59:19 module path: /home/sivukhin/projects/go/cache
2024/01/04 19:59:19 ready to compile project at path '/home/sivukhin/projects/go/cache' for assembly inspection
2024/01/04 19:59:19 ready to parse assembly output
2024/01/04 19:59:59 ready to normalize assembly lines (size 271)
2024/01/04 19:59:59 ready to analyze module AST
2024/01/04 20:00:04 it seems like your code vanished from compiled binary: func=[Create], file=[/home/sivukhin/projects/go/cache/api/controller.go], line=[58], snippet:
    return dto.SaveResponse{}, apiErr
```
