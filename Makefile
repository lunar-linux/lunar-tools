#
# Makefile for lunar-tools - a collection of lunar-related shell tools
#

# versioning scheme: since this is mostly a linear process if incremental
# but we do not update that often we use year.number as version number
# i.e. 2004.9 2004.10 2004.11 ...
VERSION = 2012.5

PROGRAMS = lids/lids luser/luser lnet/lnet lservices/lservices \
	lmodules/lmodules clad/clad ltime/ltime
DOCS = README COPYING
MANPAGES = lnet/lnet.8
PROFILEDFILES = clad/clad.rc

BINDIR = /usr/bin/
SBINDIR = /usr/sbin/
MANDIR = /usr/share/man/
DOCDIR = /usr/share/doc/lunar-tools/
PROFILEDDIR = /etc/profile.d/

all:
install:
	if [ ! -d "/sbin" ] ; then \
	    mkdir -p "/sbin" ; \
	fi
	install -m755 installkernel/installkernel /sbin/
	if [ ! -d "${SBINDIR}" ] ; then \
	    mkdir -p ${SBINDIR} ; \
	fi
	for PROGRAM in ${PROGRAMS} ; do \
	    install -m755 $${PROGRAM} ${SBINDIR}/ ; \
	done
	for MANPAGE in ${MANPAGES} ; do \
	    EXT=`echo "$${MANPAGE:(($${#MANPAGE}-1)):1}"` ; \
	    if [ ! -d "${MANDIR}man$$EXT" ] ; then \
	        mkdir -p ${MANDIR}man$$EXT ; \
	    fi ; \
	    install -m644 $${MANPAGE} ${MANDIR}man$$EXT/ ; \
	done
	if [ ! -d "${PROFILEDDIR}" ] ; then \
	    mkdir -p ${PROFILEDDIR} ; \
	fi
	for RCFILE in ${PROFILEDFILES} ; do \
	    install -m644 $${RCFILE} ${PROFILEDDIR}/ ; \
	done
	if [ ! -d "${DOCDIR}" ] ; then \
		mkdir -p ${DOCDIR} ; \
	fi
	for DOC in ${DOCS} ; do \
		install -m644 $${DOC} ${DOCDIR}/ ; \
	done

dist:
	git archive --format=tar --prefix=lunar-tools-$(VERSION)/ lunar-tools-$(VERSION) | bzip2 > lunar-tools-$(VERSION).tar.bz2
