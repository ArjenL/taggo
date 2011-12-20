taggo - Generate Exuberant Ctags-compatible tags for Go source
==============================================================

This is my tagfile generator.  There are many others like it, but this
one is mine...

This generator outputs tags for the following Go declarations:

* Toplevel variables
* Constants
* Functions and methods
* Types and interfaces

Usage
-----

<pre>
$ taggo [-recurse] file.go
</pre>

When the -recurse option is given, any subdirectories given on the
command-line are recursively searched for Go source files (.go
extension).

The generated tags are printed to the standard output.
