#!/usr/bin/env python3
"""Build Tailcat A/B candidates without changing published module references.

Only an external artifact directory is written. Local dependency replacements
are explicitly test-only; a release must use separately published modules.
"""
from __future__ import annotations
import argparse
import hashlib
import json
import os
from pathlib import Path
import shutil
import subprocess


def git(root: Path, *args: str) -> str:
    return subprocess.check_output(['git', '-C', str(root), *args], text=True).strip()


def main() -> None:
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument('--output', type=Path, required=True)
    p.add_argument('--tailscale-root', type=Path)
    p.add_argument('--quic-root', type=Path)
    p.add_argument('--platforms', default='linux/amd64,darwin/arm64')
    a = p.parse_args()
    root, out = Path.cwd().resolve(), a.output.resolve()
    if out == root or root in out.parents:
        p.error('artifact directory must be outside the checkout')
    if bool(a.tailscale_root) != bool(a.quic_root):
        p.error('set both shared dependency paths or neither for the baseline')
    platforms = a.platforms.split(',')
    if any(x not in ('linux/amd64','linux/arm64','darwin/arm64','darwin/amd64','windows/amd64','windows/arm64') for x in platforms):
        p.error('unsupported platform')
    out.mkdir(parents=True, exist_ok=True)
    mod = out/'candidate.mod'
    shutil.copyfile(root/'go.mod', mod)
    shutil.copyfile(root/'go.sum', out/'candidate.sum')
    record = {'source':git(root,'rev-parse','HEAD'),'source_dirty':bool(git(root,'status','--porcelain')),
              'release_ready':False,'instrumented':True,'local_dependencies':{},'files':{}}
    for name, source in [('tailscale.com',a.tailscale_root),('github.com/quic-go/quic-go',a.quic_root)]:
        if source:
            source=source.resolve()
            if not (source/'go.mod').is_file():p.error('dependency root is missing go.mod')
            subprocess.run(['go','mod','edit','-modfile='+str(mod),'-replace='+name+'='+str(source)],check=True)
            record['local_dependencies'][name]={'path':str(source),'commit':git(source,'rev-parse','HEAD'),'dirty':bool(git(source,'status','--porcelain'))}
    for platform in platforms:
        goos,goarch=platform.split('/')
        env={**os.environ,'GOOS':goos,'GOARCH':goarch,'CGO_ENABLED':'0'}
        for name, package in [('tailcat','./cmd/tailcat'),('wancheck','./internal/wancheck')]:
            target=out/(name+'-'+goos+'-'+goarch+('.exe' if goos=='windows' else ''))
            subprocess.run(['go','build','-mod=readonly','-modfile='+str(mod),'-tags=tailcat_perf','-trimpath','-o',str(target),package],env=env,check=True)
            record['files'][target.name]={'sha256':hashlib.sha256(target.read_bytes()).hexdigest(),'size':target.stat().st_size}
            print('BUILT',target.name,flush=True)
    (out/'build.json').write_text(json.dumps(record,indent=2)+'\n')


if __name__=='__main__':main()
