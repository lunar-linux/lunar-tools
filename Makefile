#
# Makefile for ltools - a collection of lunar-related shell tools
#

PROGRAMS = lids/lids luser/luser lnet/lnet
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

