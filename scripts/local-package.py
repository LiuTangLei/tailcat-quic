#!/usr/bin/env python3
"""Finite local builds and release archives; never invokes GitHub Actions.

Test builds may override dependencies. Release packaging requires a clean Git
checkout and immutable public module pins. Upload/publication is not automatic.
"""
from __future__ import annotations
import argparse
import gzip
import hashlib
import io
import json
import os
from pathlib import Path
import re
import shutil
import subprocess
import tarfile
import zipfile

TARGETS = ('linux/amd64', 'linux/arm64', 'linux/arm', 'darwin/amd64',
           'darwin/arm64', 'windows/amd64', 'windows/arm64')

def capture(*cmd: str) -> str:
    return subprocess.check_output(cmd, text=True).strip()

def main() -> None:
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument('--output', type=Path, required=True)
    p.add_argument('--version', required=True)
    p.add_argument('--platforms', default=','.join(TARGETS))
    p.add_argument('--test-only', action='store_true')
    p.add_argument('--tailscale-root', type=Path)
    p.add_argument('--quic-root', type=Path)
    p.add_argument('--udp-probe', action='store_true')
    p.add_argument('--linux-packages', action='store_true', help='also build .deb/.rpm locally with pinned nFPM')
    a = p.parse_args()
    if not re.fullmatch(r'v\d+\.\d+\.\d+(?:-quic\.\d+)?', a.version):
        p.error('use an ordinary version or quic.N suffix, never h3.N')
    root, out = Path.cwd().resolve(), a.output.resolve()
    if out == root or root in out.parents:
        p.error('output must be outside the checkout')
    targets = a.platforms.split(',')
    if not targets or any(x not in TARGETS for x in targets):
        p.error('unsupported platform')
    dirty = bool(capture('git', 'status', '--porcelain'))
    overrides = [('tailscale.com', a.tailscale_root), ('github.com/quic-go/quic-go', a.quic_root)]
    if not a.test_only and (dirty or any(path for _, path in overrides)):
        p.error('release requires clean source and no local dependency overrides')
    if (out / 'build.json').exists():
        p.error('output already contains a build; preserve previous evidence')
    out.mkdir(parents=True, exist_ok=True)
    mod = out / 'candidate.mod'
    shutil.copyfile(root/'go.mod', mod)
    shutil.copyfile(root/'go.sum', out/'candidate.sum')
    for name, path in overrides:
        if path:
            path = path.resolve()
            if not (path/'go.mod').is_file():
                p.error('missing dependency module')
            subprocess.run(['go', 'mod', 'edit', '-modfile='+str(mod), '-replace='+name+'='+str(path)], check=True)
    modules = {}
    for name in ('tailscale.com', 'github.com/quic-go/quic-go', 'github.com/LiuTangLei/wireguard-go'):
        module = json.loads(capture('go', 'list', '-modfile='+str(mod), '-m', '-json', name))
        selected = module.get('Replace') or module
        if not a.test_only and not selected.get('Version'):
            p.error('unversioned dependency: '+name)
        modules[name] = {k: selected[k] for k in ('Path', 'Version', 'Sum') if k in selected}
    revision = capture('git', 'rev-parse', 'HEAD')
    record = {'source': revision, 'source_dirty': dirty, 'version': a.version,
              'test_only': a.test_only, 'toolchain': capture('go', 'version'),
              'modules': modules, 'files': {}, 'archives': {}}
    tags = (root/'build-tags.txt').read_text().strip()
    for target in targets:
        goos, arch = target.split('/')
        env = {**os.environ, 'GOOS': goos, 'GOARCH': arch, 'GOARM': '7',
               'CGO_ENABLED': '0', 'GOWORK': 'off'}
        suffix = '.exe' if goos == 'windows' else ''
        binary = out/('tailcat-'+goos+'-'+arch+suffix)
        cmd = ['go','build','-mod=readonly','-modfile='+str(mod),'-trimpath',
               '-tags='+tags,'-ldflags=-s -w -X main.version='+a.version,'-o',str(binary),'./cmd/tailcat']
        subprocess.run(cmd, env=env, check=True)
        record['files'][binary.name] = hashlib.sha256(binary.read_bytes()).hexdigest()
        print('BUILT', target, record['files'][binary.name], flush=True)
        if a.udp_probe and target == 'linux/amd64':
            probe = out/'wancheck-linux-amd64'
            subprocess.run(cmd[:-2]+[str(probe), './internal/wancheck'], env=env, check=True)
            record['files'][probe.name] = hashlib.sha256(probe.read_bytes()).hexdigest()
        if a.test_only:
            continue
        docs = ['LICENSE', 'README.md', 'README.zh-CN.md', 'README.ja.md', 'INSTALL.md', 'SECURITY.md', 'THIRD_PARTY_NOTICES.md']
        validation = 'docs/release-validation-'+a.version+'.md'
        if (root/validation).is_file():
            docs.append(validation)
        members = [('tailcat'+suffix, binary.read_bytes(), 0o755)]
        members += [(name, (root/name).read_bytes(), 0o644) for name in docs]
        name = 'tailcat_'+a.version[1:]+'_'+goos+'_'+arch+('v7' if arch == 'arm' else '')
        archive = out/(name+('.zip' if goos == 'windows' else '.tar.gz'))
        if goos == 'windows':
            with zipfile.ZipFile(archive, 'w', compression=zipfile.ZIP_DEFLATED) as z:
                for filename, data, mode in members:
                    info = zipfile.ZipInfo(filename, (2020,1,1,0,0,0))
                    info.external_attr = mode << 16
                    info.compress_type = zipfile.ZIP_DEFLATED
                    z.writestr(info, data)
        else:
            with archive.open('wb') as raw, gzip.GzipFile(filename='', fileobj=raw, mode='wb', mtime=0) as compressed:
                with tarfile.open(fileobj=compressed, mode='w') as tar:
                    for filename, data, mode in members:
                        info = tarfile.TarInfo(filename)
                        info.size, info.mode, info.mtime = len(data), mode, 0
                        tar.addfile(info, io.BytesIO(data))
        record['archives'][archive.name] = hashlib.sha256(archive.read_bytes()).hexdigest()
    if a.linux_packages and not a.test_only:
        for arch, nfpm_arch in (('amd64','amd64'), ('arm64','arm64'), ('arm','arm7')):
            binary = out/('tailcat-linux-'+arch)
            if not binary.is_file():
                continue
            config = {'name':'tailcat-quic', 'arch':nfpm_arch, 'platform':'linux',
                      'version':a.version[1:], 'maintainer':'LiuTangLei',
                      'description':'Independent QUIC/HTTP3-only Tailcat tunnel with BBRv3.',
                      'homepage':'https://github.com/LiuTangLei/tailcat-quic', 'license':'BSD-3-Clause',
                      'conflicts':['tailcat','tailcat-h3'], 'replaces':['tailcat-h3'],
                      'contents':[{'src':str(binary), 'dst':'/usr/bin/tailcat', 'file_info':{'mode':493}}]}
            for doc in ('LICENSE','README.md','INSTALL.md','SECURITY.md','THIRD_PARTY_NOTICES.md'):
                config['contents'].append({'src':str(root/doc),'dst':'/usr/share/doc/tailcat-quic/'+doc,
                                            'file_info':{'mode':420}})
            config_path = out/('nfpm-'+arch+'.json')
            config_path.write_text(json.dumps(config, indent=2)+'\n')
            for fmt in ('deb','rpm'):
                package = out/('tailcat-quic_'+a.version[1:]+'_'+nfpm_arch+'.'+fmt)
                subprocess.run(['go','run','github.com/goreleaser/nfpm/v2/cmd/nfpm@v2.33.1','package',
                                '--config',str(config_path),'--packager',fmt,'--target',str(package)],
                               check=True, env={**os.environ,'GOWORK':'off'})
                record['archives'][package.name] = hashlib.sha256(package.read_bytes()).hexdigest()
    (out/'build.json').write_text(json.dumps(record, indent=2)+'\n')
    if record['archives']:
        (out/'checksums.txt').write_text(''.join(digest+'  '+name+'\n' for name,digest in sorted(record['archives'].items())))
    print('MANIFEST', out/'build.json', flush=True)

if __name__ == '__main__':
    main()
