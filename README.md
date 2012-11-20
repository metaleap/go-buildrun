go-buildrun
===========

This small and simple tool is for "slightly-augmented" integration of the **go install** command into Sublime Text 2 (or any other editor/IDE for that matter).

(As you probably know, "go install" is a neat command: it only rebuilds/relinks those dependencies that have actually changed at the source code level since their last build. Perfect for many quick fire-and-forget "build-and-run" cycles from your IDE or editor.)

1. A command-line argument **-f** is given to the tool, the .go source to "build" (go install) from. (See my included go-buildrun.sublime-build for example.)

2. As the name "buildrun" implies, it is to "build and, if applicable, run". So it checks if the current .go source file is a main package, if so, the compiled program is run immediately when "go install" returns with no errors.

However, **prior** to invoking "go install":

- IF the current package directory contains any **.gt** and **.gt.go** files, the tool applies those *.gt* template definitions to the *.gt.go* source files. See "Templating" section below.

- IF the package directory, or any of its ancestor directories (up to but not including $GOPATH) contains a *.go-buildrun* text file, it executes the command specified in that file's single line, with a single argument (remember, in your custom tool, that will be *os.Args[1]* rather than 0, which is the executable file path itself) --- the full directory path that this *.go-buildrun* text file resides in (not the executable itself, which is most likely in your $GOPATH/bin folder).


IF the command-line argument **-d** is *true*, the package is not a main package AND contains a *doc.go* source file, then upon successful build **godoc** is run to generate a single **doc.html** package documentation file in the package directory.


Templating
==========


A very simplistic minimalist "templating system". Re-generates a designated portion within a .gt.go source file based on a template specified in a .gt file.

The .gt **template provider file** is a normal Go source file, but *always* contains a **package gt** clause, anything after this clause is used for templating, anything before this clause is ignored. Your custom templating parameters are prefixed *and* suffixed by two underscores (__), example *slice.gt*:


    package gt
    type __TT__Slice []*__TT__
    func (me *__TT__Slice) Add (value *__TT__) {
        *me = append(*me, value)
    }


The .gt.go **template consumer file** is a normal Go source file that can designate *one* (1) portion within it to be re-generated based on a .gt template file and parameterize the template with the **//#begin-gt** and **//#end-gt** directives:


    package mypkg
    type MyObject struct { ID string }
    //#begin-gt slice.gt TT:MyObject
    // NOTE the following is template-generated code until the #end-gt directive!
    type MyObjectSlice []*MyObject
    func (me *MyObjectSlice) Add (value *MyObject) {
        *me = append(*me, value)
    }
    //#end-gt
    var myObjs = MyObjectSlice {}


That's it. Every time the package is re-built with *go-buildrun*, all **#begin-gt ... #end-gt** portions in all .gt.go source files are regenerated from the .gt template-file and replacement-parameters specified in their **#begin-gt** directive.
