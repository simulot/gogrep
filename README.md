# gogrep
:construction: Still under construction, but can be tested :construction:

In my daily job, I'm working on an Windows application that 
generates numerous and detailed log files. During 24h, we have 
100.000 for several gigabytes. They are zipped daily.

To search something into logs, archives must be unziped 
before using a search tool like grep or FINDSTR. This 
process need some disk space, and is very time consuming.
Furthermore, FINDSTR windows' utility doesn't handle regular 
expressions correctly, and has some nasty bugs [(see ss64.com)](http://ss64.com/nt/findstr.html). 

This was my motivation for writing this utility.


## Requirements for a tool to search into Zip archives

* faster than unzipping then search
* simple to use, one tool for searching files, folders, archives
* search in huge .zip or .tgz archive
* search in a collection of zip files
* windows and linux binaries at least
* search only in selected files inside archives
* search in plain ascii files, but also in utf-8, utf-16 encoded files

## TODO:
- [X] ascii and UTF-8 
- [X] zip content file mask
- [X] read folder archives
- [X] read zip archives
- [X] read tar.gz archives
- [X] UTF-16 reading
- [_] CSV output
- [X] colorized output ala grep (on linux)


## Faster than unzipping then search
Speed gain comes from skipping deziping file before the search..

## Faster than grep when searching across multiple files
A GO compiled program is known to be less efficient than a c program. Go REGEXP is also less efficient than c REGXEP. 
Nevertheless, goroutines allows to handle several files at a time. I get On a 16 core SSD laptop, gogrep is 2 times faster than grep. 
## Simple to use
Windows doesn't provide decent grep like tool. gogrep is self sufficient.

## Cross platform
Thanks to Go compiler, it is (really) easy to compile 
binaries for a variety of OSes and hardware. 


## Side note : 5 years later, let's refactor it!
I got an issue on github about binaries release. Someone is interested in my work!  Since the original project, my job has changed. I don't need anymore to crawl huge archives to find subtile errors in the log.

Nevertheless, I gave a look to the code, and I found it convoluted and unnecessary complex. I have now a better understanding of Go features, when use them, when not. I have also learned that less lines of code produce less errors. 

Let see I can improve that! I have refactored the code 
- to simplify it
- to use standard library instead of my own code
- to reduce memory allocations and use sync.pool
- to use concurrency just where it makes sense

And the result is indeed far better. On the same dataset, same hardware and same version of go compiler, the program runs more than 5 times faster. Not too bad, is it?
