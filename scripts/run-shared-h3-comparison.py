#!/usr/bin/env python3
"""Finite same-host Tailcat baseline/candidate tests with shared host locks.

No installation or production configuration is changed. The worker is polled
by the caller; completion requires recorded cleanup for every individual run.
"""
from __future__ import annotations
import argparse
import fcntl
import hashlib
import json
import os
from pathlib import Path
import signal
import subprocess
import sys
import tempfile
import time


def save(path, data):
    temporary=path.with_suffix('.tmp')
    temporary.write_text(json.dumps(data,indent=2)+'\n')
    temporary.replace(path)


def main():
    p=argparse.ArgumentParser(description=__doc__)
    p.add_argument('action',choices=['start','worker','status'])
    p.add_argument('--output',type=Path,required=True)
    p.add_argument('--server')
    p.add_argument('--client')
    p.add_argument('--server-address')
    p.add_argument('--client-address')
    p.add_argument('--baseline',type=Path)
    p.add_argument('--candidate',type=Path)
    p.add_argument('--rounds',type=int,default=2)
    p.add_argument('--seconds',type=int,default=10)
    p.add_argument('--size-mib',type=int,default=8)
    a=p.parse_args();out=a.output.resolve();statefile=out/'driver.json'
    if a.action=='status':print(statefile.read_text());return
    if not all((a.server,a.client,a.server_address,a.client_address,a.baseline,a.candidate)):
        p.error('explicit hosts, canonical addresses and both binaries required')
    if a.server_address==a.client_address or not 1<=a.rounds<=2 or not 3<=a.seconds<=20 or not 1<=a.size_mib<=16:
        p.error('invalid finite benchmark bounds')
    root=Path.cwd().resolve()
    if out==root or root in out.parents:p.error('output must be outside source')
    out.mkdir(parents=True,exist_ok=True)
    if a.action=='start':
        if statefile.exists():p.error('do not overwrite an existing run')
        with (out/'driver.log').open('w') as log:
            process=subprocess.Popen([sys.executable,str(Path(__file__).resolve()),'worker',*sys.argv[2:]],stdin=subprocess.DEVNULL,stdout=log,stderr=log,start_new_session=True)
        print(json.dumps({'pid':process.pid,'state':str(statefile)}));return
    state={'pid':os.getpid(),'status':'running','passed':False,'runs':[],'started':time.time()}
    save(statefile,state)
    child=None;locks=[]
    def cancel(signum,frame):
        if child and child.poll() is None:child.terminate()
        raise InterruptedError('comparison interrupted')
    signal.signal(signal.SIGTERM,cancel);signal.signal(signal.SIGINT,cancel)
    try:
        for address in sorted((a.server_address,a.client_address)):
            name=hashlib.sha256(address.encode()).hexdigest()[:20]
            lock=open(Path(tempfile.gettempdir())/('tailscale-transport-'+name+'.lock'),'a+')
            limit=time.monotonic()+600
            while True:
                try:fcntl.flock(lock,fcntl.LOCK_EX|fcntl.LOCK_NB);locks.append(lock);break
                except BlockingIOError:
                    if time.monotonic()>limit:raise RuntimeError('host is in use by another test')
                    time.sleep(.5)
        for round_number in range(1,a.rounds+1):
            # Alternate the order so the candidate is not always measured later.
            order=[('baseline',a.baseline),('candidate',a.candidate)]
            if round_number%2==0:order.reverse()
            for kind,binary in order:
                report=out/f'{kind}-{round_number}.json'
                entry={'kind':kind,'round':round_number,'status':'running','report':str(report)}
                state['runs'].append(entry);save(statefile,state)
                command=[sys.executable,str(Path(__file__).with_name('host_pair_probe.py')),'drive','--server',a.server,'--client',a.client,'--server-label','A','--client-label','B','--binary',str(binary.resolve()),'--seconds',str(a.seconds),'--size-mib',str(a.size_mib),'--report',str(report)]
                with (out/f'{kind}-{round_number}.log').open('w') as log:
                    child=subprocess.Popen(command,stdin=subprocess.DEVNULL,stdout=log,stderr=log)
                    try:code=child.wait(timeout=480)
                    except subprocess.TimeoutExpired:
                        child.terminate()
                        try:child.wait(timeout=45)
                        except subprocess.TimeoutExpired:child.kill();child.wait()
                        raise RuntimeError('bounded Tailcat test timed out')
                entry['returncode']=code
                if not report.is_file():raise RuntimeError('missing Tailcat test report: '+str(report))
                data=json.loads(report.read_text())
                expected=hashlib.sha256(binary.read_bytes()).hexdigest()
                entry.update(status='finished',passed=code==0 and bool(data.get('ok')) and not data.get('cleanup_errors') and data.get('binary_sha256')==expected)
                save(statefile,state)
                print('FINISHED',kind,round_number,entry['passed'],flush=True)
                if not entry['passed']:raise RuntimeError('functional/cleanup failure: '+str(report))
        state.update(status='finished',passed=all(x['passed'] for x in state['runs']))
    except Exception as exc:state.update(status='failed',error=str(exc))
    finally:
        if child and child.poll() is None:
            child.terminate()
            try:child.wait(timeout=45)
            except subprocess.TimeoutExpired:child.kill();child.wait()
        state['seconds']=round(time.time()-state['started'],2);save(statefile,state)
    if not state['passed']:raise SystemExit(1)


if __name__=='__main__':main()
