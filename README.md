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


## EDIT: 5 years later, let's refactor it
I got an issue on github about binarie release. Somone is interested in my work!  Since the original project, my job has changed. I don't need anymore to crawl huge archives to find subtil errors in the log.

Nevertheless, I gave a look to the code, and I found it convoluted and unnecessary complex. I have now a better understanding of Go features, when use them, when not. I have also learned that less lines of code produce less errors. 

Let see I can improve that! I have refactored the code 
- to get rid of complexity 
- to use standard library instead of my own code
- to reduce memory allocations and use sync.pool
- to use concurency just where it make sense

And the result is indeed far better. On the same dataset, same hardware and same version of go compiler, the program run more than 5 times faster. Not too bad, is it?

The original version takes 6.3s to read the data set.

``` sh
$ time ./gogrep  ORA-[0-9]\{5\} ../DATA/LogFiles
ImportInvoiceBT__SVC001_20161117100102.log:10346:17/11/2016 10:01:14.602 [ERROR  ]ORA-00001: unique constraint (BASW.EXT_BT_ACK_STATUS_PK) violated
ImportInvoiceBT__SVC001_20161117094102.log:12502:17/11/2016 10:01:14.633 [ERROR  ]ORA-00001: unique constraint (BASW.EXT_BT_ACK_STATUS_PK) violated

real    0m6,386s
user    0m48,467s
sys     0m3,932s
j
```

When the new version takes only 1.725s, yes 3.7 faster! CPU utilization show a sharper peek denoting full utilisation of all CPU cores

```sh
$ time ./gogrep ORA-[0-9]\{5\} ../tests/gogrep/DATA/LogFiles
ImportInvoiceBT__SVC001_20161117100102.log(10346):17/11/2016 10:01:14.602 [ERROR  ]ORA-00001: unique constraint (BASW.EXT_BT_ACK_STATUS_PK) violated
ImportInvoiceBT__SVC001_20161117094102.log(12502):17/11/2016 10:01:14.633 [ERROR  ]ORA-00001: unique constraint (BASW.EXT_BT_ACK_STATUS_PK) violated


real    0m1,725s
user    0m17,157s
sys     0m1,442s
```

Well, how it is comparing to grep itself?


```
$ time grep -r "ORA-[0-9]\{5\}" ../tests/gogrep/DATA/LogFiles
ImportInvoiceBT__SVC001_20161117094102.log:17/11/2016 10:01:14.633 [ERROR  ]ORA-00001: unique constraint (BASW.EXT_BT_ACK_STATUS_PK) violated
ImportInvoiceBT__SVC001_20161117100102.log:17/11/2016 10:01:14.602 [ERROR  ]ORA-00001: unique constraint (BASW.EXT_BT_ACK_STATUS_PK) violated

real    0m3,585s
user    0m2,639s
sys     0m0,943s
```

2 times faster than grep! 💪
