#
# Makefile for ltools - a collection of lunar-related shell tools
#

# versioning scheme: since this is mostly a linear process if incremental
# but we do not update that often we use year.number as version number
# i.e. 2004.9 2004.10 2004.11 ...
VERSION = 2004.1

PROGRAMS = lids/lids luser/luser lnet/lnet lservices/lservices
SBINDIR = /usr/sbin/
BINDIR = /usr/bin/
MANDIR = /usr/share/man/
MANPAGES = 

all:
install:
	mkdir -p ${SBINDIR} ;
	for PROGRAM in ${PROGRAMS} ; do \
	    install -m755 $${PROGRAM} ${SBINDIR} ; \
	done ;

release:
	cd .. ; \
	tar cjvf /tmp/ltools-${VERSION}.tar.bz2 --exclude="*/CVS*" ltools/ ; \
	cd - ;
