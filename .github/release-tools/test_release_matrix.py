import argparse
import hashlib
import importlib.util
import io
import json
import os
from pathlib import Path
import shutil
import subprocess
import tarfile
import tempfile
import unittest
from unittest.mock import patch

import yaml

ROOT = Path(__file__).resolve().parents[2]
BASH = str(Path(shutil.which('git')).parent.parent / 'bin/bash.exe') if os.name == 'nt' else 'bash'
spec = importlib.util.spec_from_file_location('release_matrix', Path(__file__).with_name('release_matrix.py'))
release = importlib.util.module_from_spec(spec)
spec.loader.exec_module(release)


class ReleaseMatrixTest(unittest.TestCase):
    def setUp(self):
        self.previous = Path.cwd()
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        os.chdir(self.temp.name)
        self.addCleanup(os.chdir, self.previous)
        shutil.copyfile(ROOT / '.goreleaser.yaml', '.goreleaser.yaml')
        Path('backend/cmd/server').mkdir(parents=True)
        release.VERSION_FILE.write_text('9.8.7.1\n')

    def fixture_artifacts(self, simple=False):
        directory = Path('release-input')
        directory.mkdir()
        for target in release.targets(simple):
            name = release.archive_name('9.8.7.1', target)
            archive = directory / name
            if target['goos'] == 'linux':
                with tarfile.open(archive, 'w:gz') as out:
                    info = tarfile.TarInfo('sub2api')
                    info.size = 7
                    info.mode = 0o755
                    out.addfile(info, io.BytesIO(b'fixture'))
            else:
                archive.write_bytes(b'fixture archive')
            metadata = {'version': '9.8.7.1', 'sha': 'a' * 40, 'target': target,
                        'archive': name, 'sha256': release.sha256(archive)}
            (directory / f"manifest-{target['goos']}-{target['goarch']}.json").write_text(json.dumps(metadata))
        return argparse.Namespace(input='release-input', version='9.8.7.1', sha='a' * 40, simple=simple, output='contexts')

    def test_full_and_simple_matrix_match_existing_targets(self):
        full = release.targets()
        self.assertEqual(full, [{'goos': 'linux', 'goarch': 'amd64'}])
        self.assertEqual(release.targets(True), full)
        self.assertEqual(release.archive_name('9.8.7.1', full[0]), 'sub2api_9.8.7.1_linux_amd64.tar.gz')

    def test_leaf_keeps_packaging_and_selects_only_one_target(self):
        original = release.config()
        release.generate_config(argparse.Namespace(mode='build', simple=False, goos='linux', goarch='amd64', output='leaf.yaml'))
        leaf = yaml.safe_load(Path('leaf.yaml').read_text(encoding='utf-8'))
        self.assertEqual(leaf['builds'][0]['goos'], ['linux'])
        self.assertEqual(leaf['builds'][0]['goarch'], ['amd64'])
        self.assertEqual(leaf['builds'][0]['ignore'], [])
        self.assertEqual(leaf['archives'], original['archives'])
        self.assertEqual(leaf['release'], original['release'])
        self.assertFalse(leaf['dockers'])
        self.assertIn('{{ .Env.RELEASE_DATE }}', '\n'.join(leaf['builds'][0]['ldflags']))
        with self.assertRaisesRegex(ValueError, 'unsupported build target'):
            release.generate_config(argparse.Namespace(mode='build', simple=False, goos='linux', goarch='arm64', output='leaf.yaml'))

    def test_publication_config_has_no_compilation_or_docker_work(self):
        for simple in (False, True):
            with self.subTest(simple=simple):
                original = release.config(simple)
                release.generate_config(argparse.Namespace(mode='publish', simple=simple, output='publisher.yaml'))
                data = yaml.safe_load(Path('publisher.yaml').read_text(encoding='utf-8'))
                self.assertTrue(data['builds'][0]['skip'])
                self.assertFalse(data['archives'])
                self.assertFalse(data['dockers'])
                self.assertEqual(data['release']['header'], original['release']['header'])
                self.assertEqual(data['release']['footer'], original['release']['footer'])
                if simple:
                    self.assertTrue(data['checksum']['disable'])
                    self.assertTrue(data['release']['skip_upload'])
                else:
                    self.assertEqual(data['checksum']['extra_files'], data['release']['extra_files'])

    def test_collect_and_verify_hash_and_source_binding(self):
        args = self.fixture_artifacts()
        release.verify(args)
        file = next(Path(args.input).glob('*.tar.gz'))
        file.write_bytes(b'corrupted')
        with self.assertRaisesRegex(ValueError, 'checksum mismatch'):
            release.verify(args)

    def test_missing_extra_and_wrong_commit_artifacts_are_rejected(self):
        args = self.fixture_artifacts(True)
        args.sha = 'b' * 40
        with self.assertRaises(ValueError):
            release.verify(args)
        args.sha = 'a' * 40
        Path('release-input/unexpected').write_text('not an asset')
        with self.assertRaises(ValueError):
            release.verify(args)
        Path('release-input/unexpected').unlink()
        next(Path('release-input').glob('*.tar.gz')).unlink()
        with self.assertRaises(FileNotFoundError):
            release.verify(args)

    def test_linux_context_preserves_binary_executable_mode(self):
        args = self.fixture_artifacts()
        Path('Dockerfile.goreleaser').write_text('FROM scratch\nCOPY sub2api /sub2api\n')
        Path('deploy').mkdir()
        Path('deploy/docker-entrypoint.sh').write_text('#!/bin/sh\nexec /app/sub2api\n')
        Path('backend/resources').mkdir()
        Path('backend/resources/data').write_text('fixture')
        release.contexts(args)
        for arch in ('amd64',):
            binary = Path('contexts') / arch / 'sub2api'
            self.assertEqual(binary.read_bytes(), b'fixture')
            if os.name != 'nt':
                self.assertEqual(binary.stat().st_mode & 0o777, 0o755)
        self.assertFalse((Path('contexts') / 'arm64').exists())

    def test_plan_requires_a_tag_for_publication(self):
        args = argparse.Namespace(ref='main', dry_run=False, simple=False)
        with patch.object(subprocess, 'check_output', return_value='a' * 40 + '\n'):
            with self.assertRaisesRegex(ValueError, 'version tag'):
                release.plan(args)
        args.ref = 'v9.8.7'
        with patch.object(subprocess, 'check_output', return_value='a' * 40 + '\n'):
            with self.assertRaisesRegex(ValueError, 'version tag'):
                release.plan(args)
        args.ref = 'v9.8.7.1'
        with patch.object(subprocess, 'check_output', side_effect=['a' * 40 + '\n', 'b' * 40 + '\n']):
            with self.assertRaisesRegex(ValueError, 'does not match'):
                release.plan(args)

    def test_published_tag_uses_four_part_version(self):
        with patch.dict(os.environ, {'GITHUB_OUTPUT': 'outputs'}), patch.object(subprocess, 'check_output', side_effect=['a' * 40 + '\n'] * 2):
            release.plan(argparse.Namespace(ref='v9.8.7.1', dry_run=False, simple=False))
        output = dict(line.split('=', 1) for line in Path('outputs').read_text().splitlines())
        self.assertEqual(output['version'], '9.8.7.1')
        self.assertEqual(json.loads(output['matrix'])['include'], [{'goos': 'linux', 'goarch': 'amd64'}])
        self.assertEqual(release.VERSION_FILE.read_text(), '9.8.7.1\n')

    def test_dry_run_plan_resolves_matrix_without_a_new_tag(self):
        with patch.dict(os.environ, {'GITHUB_OUTPUT': 'outputs', 'GITHUB_REPOSITORY_OWNER': 'ExampleOwner'}), patch.object(subprocess, 'check_output', return_value='a' * 40 + '\n'):
            release.plan(argparse.Namespace(ref='feature/matrix', dry_run=True, simple=False))
        output = dict(line.split('=', 1) for line in Path('outputs').read_text().splitlines())
        self.assertEqual(output['dry_run'], 'true')
        self.assertEqual(output['owner_lower'], 'exampleowner')
        self.assertEqual(output['version'], '9.8.7.1')
        self.assertEqual(json.loads(output['matrix'])['include'], [{'goos': 'linux', 'goarch': 'amd64'}])

    def test_docker_commands_do_not_publish_during_dry_run(self):
        fake_bin = Path('bin')
        fake_bin.mkdir()
        docker = fake_bin / 'docker'
        docker.write_bytes(b'#!/bin/sh\nprintf "%s\\n" "$*" >> "$DOCKER_LOG"\n')
        docker.chmod(0o755)
        for simple in (False, True):
            with self.subTest(simple=simple):
                log_path = Path(f'docker-dry-{simple}.log').resolve()
                env = {**os.environ, 'PATH': str(fake_bin.resolve()) + os.pathsep + os.environ['PATH'],
                       'DOCKER_LOG': str(log_path), 'RUNNER_TEMP': self.temp.name,
                       'RELEASE_VERSION': '9.8.7.1', 'RELEASE_SHA': 'a' * 40, 'GITHUB_REPOSITORY': 'ExampleOwner/sub2api',
                       'DRY_RUN': 'true', 'SIMPLE_RELEASE': str(simple).lower(), 'DOCKERHUB_USERNAME': 'skip'}
                subprocess.run([BASH, str(ROOT / '.github/release-tools/release-images.sh')], env=env, check=True)
                log = log_path.read_text()
                self.assertEqual(log.count('buildx build'), 1)
                self.assertIn('linux/amd64', log)
                self.assertNotIn('arm64', log)
                self.assertNotIn('--push', log)
                self.assertNotIn('imagetools', log)
                self.assertNotIn('skip/sub2api', log)
                self.assertIn('ghcr.io/exampleowner/sub2api', log)


    def test_published_full_and_simple_image_tags(self):
        fake_bin = Path('bin')
        fake_bin.mkdir()
        docker = fake_bin / 'docker'
        docker.write_bytes(b'#!/bin/sh\nprintf "%s\\n" "$*" >> "$DOCKER_LOG"\n')
        docker.chmod(0o755)
        for simple, hub in ((False, 'fixturehub'), (False, ''), (True, 'fixturehub')):
            with self.subTest(simple=simple, hub=hub):
                log_path = Path(f'docker-{simple}-{hub}.log').resolve()
                env = {**os.environ, 'PATH': str(fake_bin.resolve()) + os.pathsep + os.environ['PATH'],
                       'DOCKER_LOG': str(log_path), 'RUNNER_TEMP': self.temp.name,
                        'RELEASE_VERSION': '9.8.7.1', 'RELEASE_SHA': 'a' * 40, 'GITHUB_REPOSITORY': 'ExampleOwner/sub2api',
                        'DRY_RUN': 'false', 'SIMPLE_RELEASE': str(simple).lower(), 'DOCKERHUB_USERNAME': hub}
                subprocess.run([BASH, str(ROOT / '.github/release-tools/release-images.sh')], env=env, check=True)
                log = log_path.read_text()
                self.assertIn('--push', log)
                self.assertEqual(log.count('buildx build'), 1)
                self.assertNotIn('arm64', log)
                self.assertNotIn('imagetools', log)
                self.assertIn('ghcr.io/exampleowner/sub2api:9.8.7.1', log)
                if simple:
                    self.assertNotIn('fixturehub', log)
                    self.assertIn('ghcr.io/exampleowner/sub2api:latest', log)
                elif hub:
                    self.assertIn('fixturehub/sub2api:9.8', log)
                    self.assertIn('ghcr.io/exampleowner/sub2api:9', log)
                else:
                    self.assertNotIn('--tag /sub2api', log)

    def test_workflow_keeps_manual_tag_and_publication_boundaries(self):
        workflow = yaml.safe_load((ROOT / '.github/workflows/release.yml').read_text(encoding='utf-8'))
        self.assertEqual(list(workflow[True]), ['workflow_dispatch'])
        steps = {step.get('name'): step for step in workflow['jobs']['release']['steps']}
        for name in ('Prepare publication-only GoReleaser config', 'Publish existing archives and release notes'):
            self.assertIn("env.SIMPLE_RELEASE != 'true' || env.DRY_RUN == 'true'", steps[name]['if'])
        self.assertIn(',publish,announce', steps['Publish existing archives and release notes']['with']['args'])
        self.assertIn("needs.prepare.outputs.dry_run != 'true'", workflow['jobs']['sync-version-file']['if'])
        sync = workflow['jobs']['sync-version-file']['steps']
        self.assertTrue(any('merge-base --is-ancestor' in step.get('run', '') for step in sync))



if __name__ == '__main__':
    unittest.main()
