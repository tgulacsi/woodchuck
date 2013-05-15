#!/usr/bin/env python

HOST = '10.65.25.51'
PORT = 12201

import os
import sys

try:
    import json
    json_escape = json.dumps
except ImportError:
    import re
    ESCAPE = re.compile(u'[\\x00-\\x1f\\\\"\\b\\f\\n\\r\\t\u2028\u2029]')
    ESCAPE_ASCII = re.compile(r'([\\"]|[^\ -~])')
    HAS_UTF8 = re.compile(r'[\x80-\xff]')
    ESCAPE_DCT = {'\\': '\\\\', '"': '\\"', '\b': '\\b', '\f': '\\f',
        '\n': '\\n', '\r': '\\r', '\t': '\\t'}
    for i in range(0x20):
        #ESCAPE_DCT.setdefault(chr(i), '\\u{0:04x}'.format(i))
        ESCAPE_DCT.setdefault(chr(i), '\\u%04x' % (i,))
    for i in [0x2028, 0x2029]:
        ESCAPE_DCT.setdefault(unichr(i), '\\u%04x' % (i,))

    FLOAT_REPR = repr

    def json_escape(s):
        """Return a JSON representation of a Python string"""
        if isinstance(s, str) and HAS_UTF8.search(s) is not None:
            s = s.decode('utf-8')

        def replace(match):
            return ESCAPE_DCT[match.group(0)]
        return u'"' + ESCAPE.sub(replace, s) + u'"'


def box_into(fileobj, params, stdin, verbose=False):
    import gzip
    gz = gzip.GzipFile(mode='w', compresslevel=2, fileobj=fileobj)
    write = ((lambda d: sys.stdout.write(d) or gz.write(d)) if verbose
            else gz.write)
    clean = None
    if params is not None:
        write('{')
        for k, v in params.iteritems():
            write('"' + k + '":')
            write((json_escape(v) if isinstance(v, basestring)
                    else str(v)) + ',')
        write('"full_message":"')
        clean = lambda a: json_escape(a[1:-1])
    if not stdin.closed:
        while 1:
            chunk = sys.stdin.read(65536)
            if not chunk:
                break
            if clean:
                chunk = clean(chunk)
            write(chunk)
    if params is not None:
        write('"}')
    gz.close()


facility_prefix = 'unosoft.glog'
h = os.environ['HOME'][6:].rstrip('/')
if os.environ['HOME'].startswith('/home/') and '/' in h:
    facility_prefix = h.replace('/', '.')

import logging
from optparse import OptionParser

op = OptionParser()
op.add_option('-v', '--verbose', action='store_true', default=False)
op.add_option('-H', '--host', type='str', default=HOST)
op.add_option('-P', '--port', type='int', default=PORT)
op.add_option('-L', '--level', type='str', default='info')
op.add_option('-F', '--facility', default='')
op.add_option('--http', type='str', default='')
op.add_option('--tcp', type='str', default='')
opts, args = op.parse_args()

name = facility_prefix + (('.' + opts.facility) if opts.facility else '')
lvl, slvl = {'critical': (logging.CRITICAL, 0), 'error': (logging.ERROR, 3),
        'warning': (logging.WARNING, 4), 'info': (logging.INFO, 6),
        'debug': (logging.DEBUG, 7)}[opts.level.lower()]

if opts.tcp or opts.http:
    import time
    params = {'version': "1.0",
        'host': os.uname()[1], 'short_message': ' '.join(args),
        'timestamp': time.time(), 'level': slvl, 'facility': name,
        'file': '', 'line': 0}
if opts.tcp:
    import socket
    sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    sock.connect((opts.tcp.split(':')[0], int(opts.tcp.split(':')[1])))
    box_into(sock.makefile(), params, sys.stdin, verbose=opts.verbose)
    sock.close()
elif opts.http:
    if not sys.stdin.closed:
        from cStringIO import StringIO
        data = StringIO()
        box_into(data, None, sys.stdin, verbose=opts.verbose)
        data = data.getvalue()
    import urllib
    r = urllib.urlopen(opts.http + '?' + urllib.urlencode(params), data)
    print r.info()
    print r.read()
else:
    import graypy
    logging.basicConfig(level=logging.INFO)
    LOG = logging.getLogger(name)
    LOG.addHandler(graypy.GELFHandler(opts.host, opts.port, fqdn=True,
            debugging_fields=False))
    if lvl < logging.INFO:
        LOG.setLevel(lvl)
    kwds = {'extra':
        {'full_message': '' if sys.stdin.closed else sys.stdin.read()}}
    logging.debug('calling %s.log(%d, *%r, **%r)', LOG, lvl, args, kwds)
    LOG.log(lvl, ' '.join('%s' * len(args)), *args, **kwds)
