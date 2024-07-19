# git-reader

A POC of a git repository structure reader.

## Usage

### Dump current HEAD reference

```sh
$ REPOSITORY=$HOME/tmp/rust ./git-reader -current
tree c6128f81e8e62eb9bd88e1c4dc9159745ffed5f2
parent ff4b39867e3033864315bf3cada039e92a6b751e
parent 633f41de0903efb830753e2e373ac9666230eb54
author bors <bors@rust-lang.org> 1721423268 +0000
committer bors <bors@rust-lang.org> 1721423268 +0000

Auto merge of #127968 - fmease:upd-jsondocck-directive-style, r=GuillaumeGomez

...
```

### Dump an object

`git-reader` can dump tree, commit, blob as well as packed by reference delta, by offset delta objects.

```sh
$ REPOSITORY=$HOME/tmp/rust ./git-reader c6128f81e8e62eb9bd88e1c4dc9159745ffed5f2
040000 tree ba4231c3c2b8dd714d8635c53f1ec5eeba2d0eb1   .github
040000 tree a7ef2e9b1f7a76af9d784b7151ebd8934b2c3b15   .reuse
040000 tree 57d1337d1cb766bbaef4a47d815863c224cacec4   LICENSES
040000 tree 77407114de14b2b5691ff1fd3d1d06a2ffba093f   compiler
040000 tree f7b1c1a6ea02a767f4aba9cdb04fc96108f35bb5   library
040000 tree 297e471fcacf8ca5a5eaed542cdbdf45166f8820   src
040000 tree 9f851ce9534cb262d3b664120491e1c82308c0da   tests
...
```

### List all references

```sh
$ REPOSITORY=$HOME/tmp/rust ./git-reader | head
cb128fad4cf72aa5e882d1c65f38bc1b9018e5eb
d50ec9daf0cbb4440737cfaf1064b95beea64734
3c3caeaf503def07daf15ba575fe6dd139cde98e
585cfbb90e40c1d3cfcd5d1718b0f77d7765d1c3
afdc6d9f03698a835cdf37e6fe3bb4c25402e545
cbf341a626ca1578e36240333c1b4d7a087f7204
e4eb0f2c97eb9d5bf77f276155e17804d934ed82
cab4f53fb22182426967dec2594f1690d37ec42e

$ REPOSITORY=$HOME/tmp/rust ./git-reader | wc -l
2651904
```

## Limitations

`git-reader` does not handle large pack files (> 2 GB). Therefore, it won't work against large clone repositories unless reducing pack files. One way to do that:

```sh
$ cd ~/tmp/linux/
$ mkdir /tmp/packs
$ ls -l .git/objects/pack
.r--r--r-- 355M mycroft 20 Jul 09:01 pack-751ca5362e3f5135be020b18a9e1f6d5dff31e86.idx
.r--r--r-- 5.3G mycroft 20 Jul 09:01 pack-751ca5362e3f5135be020b18a9e1f6d5dff31e86.pack
.r--r--r--  41M mycroft 20 Jul 09:02 pack-751ca5362e3f5135be020b18a9e1f6d5dff31e86.rev

$ mv .git/objects/pack/* /tmp/packs
$ git unpack-objects < /tmp/packs/pack-abcdef.pack
... this will take a while ...
Unpacking objects: 100% (10321986/10321986), 4.89 GiB | 2.29 MiB/s, done.

$ git repack --max-pack-size 500m
Enumerating objects: 10321986, done.
Counting objects: 100% (10321986/10321986), done.
Delta compression using up to 24 threads
Compressing objects: 100% (10265955/10265955), done.
Writing objects: 100% (10321986/10321986), done.
Total 10321986 (delta 8517928), reused 0 (delta 0), pack-reused 0 (from 0)

$ eza -l .git/objects/pack/
.r--r--r-- 104M mycroft 20 Jul 12:03 pack-0aaa8b16203da718d0211b3d5aff9229e05351ba.idx
.r--r--r-- 524M mycroft 20 Jul 12:00 pack-0aaa8b16203da718d0211b3d5aff9229e05351ba.pack
.r--r--r--  15M mycroft 20 Jul 12:03 pack-0aaa8b16203da718d0211b3d5aff9229e05351ba.rev


```
