# gogrep
:construction:NOT WORKING YET:construction:

In my daily job, I'm working on an Windows application that 
generates numerous and detailed log files. During 24h, we have 
100 thousands of files, for several gigabytes. They are zipped daily.

To search something into logs, archives must be unziped 
before using a search tool like grep or FINDSTR. This 
process need some disk space, and is very time consuming.
Furthermore, FINDSTR windows' utility doesn't handle regular 
expressions correctly, and has some nasty bugs [(see ss64.com)](http://ss64.com/nt/findstr.html). 

This was my motivation for writing this utility.

## Requirements for a tool to search into Zip archives

* faster than unzipping then search
* simple to use, one tool for searching files, folders, archives
* search in huge .zip
* search in a collection of zip files
* windows and linux binaries at least
* search only in selected files inside archives
* search in plain ascii files, but also in utf-8, utf-16 encoded files
* search occurrence of several pattern in files. 
Example output files having string "ERROR" and and a given GUID

And, nice to have features:
* CSV output
* Excel output, why not?
* Read tgz files,
* Read tar files
* colorized output

## TODO:
- [X] ascii and UTF-8 
- [X] zip content file mask
- [X] UTF-16 reading
- [ ] CSV output
- [X] colorized output ala grep


 
### Faster than unzipping then search
Grep is fast! 5 seconds on my testing file set. So, I don't expect 
to beat it on pure performance area. 

Speed gain comes from skipping intermediate file storage.


### Simple to use
No need to implement the rich palette of grep options. Let go
 strait to the point: search pattern in archived files.

### Cross platform
Thanks to Go compiler, it is (really) easy to compile 
binaries for a variety of OSes and hardware. 

# Performances

## Standard tools

Test set|Command|Duration|Remark
--------|-------|--------|-------
zipped archive | ```unzip -d temp zipfile.zip && grep```|5m2s |
zipped archive | ```for file in $1; do unzip -c "$file" \| grep -a "ORA-[[:digit:]]\{5\}"; done```| 1m22 | Pretty fast, but the name of file is lost
zipped archive | ```zipgrep ....``` | more than 2h | 
dezipped archive  | ```grep -r -a "ORA-[[:digit:]]\{5\}" folder``` | 1m4s

## With gogrep

Test set | Command | Duration | Remark
---------|---------|----------|-------- 
zipped archive  | ```gogrep ORA-\\d\{5\} zipfile.zip``` | 1m39s | Name of archived file is visible
dezipped archive | ```gogrep ORA-\\d\{5\} folder``` | 2m5s | Same command for folders and zipped files

