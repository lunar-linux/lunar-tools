#
# Makefile for lunar-tools - a collection of lunar-related shell tools
#

# versioning scheme: since this is mostly a linear process if incremental
# but we do not update that often we use year.number as version number
# i.e. 2004.9 2004.10 2004.11 ...
VERSION = 2004.1

PROGRAMS = lids/lids luser/luser lnet/lnet lservices/lservices
SBINDIR = /usr/sbin/
BINDIR = /usr/bin/
MANDIR = /usr/share/man/
MANPAGES = lnet/lnet.8

all:
install:
	for PROGRAM in ${PROGRAMS} ; do \
	    if [ ! -d "${SBINDIR}" ] ; then \
	        mkdir -p ${SBINDIR} ; \
	    fi ; \
	    install -m755 $${PROGRAM} ${SBINDIR} ; \
	done ; \
	for MANPAGE in ${MANPAGES} ; do \
	    EXT=`echo "$${MANPAGE:(($${#MANPAGE}-1)):1}"` ; \
	    if [ ! -d "${MANDIR}man$$EXT" ] ; then \
	        mkdir -p ${MANDIR}man$$EXT ; \
	    fi ; \
	    install -m644 $${MANPAGE} ${MANDIR}man$$EXT/ ; \
	done

release:
	tar cjvf /tmp/lunar-tools-${VERSION}.tar.bz2 --exclude="*/CVS*" -C .. lunar-tools/ ; \
