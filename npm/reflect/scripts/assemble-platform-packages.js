#!/usr/bin/env node
// Assemble the six per-platform npm packages prior to `npm publish`.
//
// STATUS: nothing is published to npm. Every package.json here is marked
// "private": true so `npm publish` refuses, and no workflow publishes them.
// See npm/README.md for what a real publish needs.
//
// For each supported (platform, arch) target this:
//   1. Brotli-compresses the built Reflect binary into
//      `../reflect-<platform>-<arch>/bin/<bin>.br`
//   2. Stamps the sub-package's version with the release version
//   3. Copies the third-party notices file into the sub-package
// and finally stamps the meta package (`@graycodeai/reflect`) version and its
// `optionalDependencies` pins with the same version, so npm resolves the
// matching platform package.
//
// The release version comes only from VERSION. REFLECT_NPM_VERSION, if set,
// must match it (release.yml also requires the tag to match).
//
// Why brotli? npm's tarball ceiling is ~200 MB. Brotli at max quality shrinks
// the ~60 MB stripped Go binary substantially and is decoded by Node's
// built-in zlib.brotliDecompressSync (no native deps required).
//
// Source paths come from environment variables (set in CI) and fall back to
// the GoReleaser v2 build output dirs for local testing.
//
// Reflect's binary is the plain Go binary named `reflect` (or `reflect.exe`), produced
// by GoReleaser (`.goreleaser.yml`, main `./cmd/reflect`, project_name `reflect`).
const fs = require('fs');
const path = require('path');
const { promisify } = require('util');
const zlib = require('zlib');

const brotliCompress = promisify(zlib.brotliCompress);

// GoReleaser v2 default build output root. Override with DIST_ROOT when CI
// produces artifacts elsewhere (e.g. a GITHUB_WORKSPACE-relative `dist`).
const distRoot = process.env.DIST_ROOT || path.resolve(__dirname, '..', '..', '..', 'dist');
const npmRoot = path.resolve(__dirname, '..', '..');

// Third-party notices: best-effort. Point at the canonical notices file if it
// exists; if the repo has none, copy is skipped so packaging proceeds.
const NOTICES_SOURCE = process.env.REFLECT_THIRD_PARTY_NOTICES
    || path.resolve(npmRoot, '..', '..', 'THIRD_PARTY_NOTICES.md');
const NOTICES_NAME = 'THIRD_PARTY_NOTICES.md';

const META_PKG_JSON = path.resolve(__dirname, '..', 'package.json');
const REPO_VERSION_FILE = path.resolve(npmRoot, '..', 'VERSION');

function releaseVersion() {
    const raw = fs.readFileSync(REPO_VERSION_FILE, 'utf8');
    const version = raw.trim().replace(/^v/, '');
    if (!/^\d+\.\d+\.\d+(-[0-9A-Za-z.-]+)?$/.test(version)) {
        throw new Error(`invalid release version ${JSON.stringify(raw.trim())} (want MAJOR.MINOR.PATCH)`);
    }
    if (process.env.REFLECT_NPM_VERSION && process.env.REFLECT_NPM_VERSION.replace(/^v/, '') !== version) {
        throw new Error('REFLECT_NPM_VERSION must match VERSION');
    }
    return version;
}

const VERSION = releaseVersion();

function ensureDir(p) { fs.mkdirSync(path.dirname(p), { recursive: true }); }

async function packPlatform({ platform, arch, envVar, defaultSource, binName }) {
    const pkgDir = path.join(npmRoot, `reflect-${platform}-${arch}`);
    const pkgJsonPath = path.join(pkgDir, 'package.json');

    if (!fs.existsSync(pkgJsonPath)) {
        console.error(`[assemble] Missing per-platform package at ${pkgDir}`);
        return false;
    }

    const source = process.env[envVar] || defaultSource;
    if (!fs.existsSync(source)) {
        console.error(`[assemble] Missing binary for ${platform}-${arch}: ${source}`);
        console.error(`            Set ${envVar} or build to the default location.`);
        return false;
    }

    // Stamp the sub-package's version to match the meta package.
    const subPkg = JSON.parse(fs.readFileSync(pkgJsonPath, 'utf8'));
    subPkg.version = VERSION;
    fs.writeFileSync(pkgJsonPath, JSON.stringify(subPkg, null, 4) + '\n');

    if (fs.existsSync(NOTICES_SOURCE)) {
        fs.copyFileSync(NOTICES_SOURCE, path.join(pkgDir, NOTICES_NAME));
    } else {
        console.warn(`[assemble] ${platform}-${arch}: third-party notices not found at ${NOTICES_SOURCE} (skipping)`);
    }

    // Brotli-compress into the sub-package's bin/.
    const outBr = path.join(pkgDir, 'bin', `${binName}.br`);
    ensureDir(outBr);
    const raw = fs.readFileSync(source);
    const compressed = await brotliCompress(raw, {
        params: { [zlib.constants.BROTLI_PARAM_QUALITY]: zlib.constants.BROTLI_MAX_QUALITY },
    });
    fs.writeFileSync(outBr, compressed);
    console.log(
        `[assemble] reflect-${platform}-${arch}@${VERSION}: ` +
        `${(raw.length / 1048576).toFixed(1)} MB -> ${(compressed.length / 1048576).toFixed(1)} MB ` +
        `(${path.relative(npmRoot, outBr)})`
    );
    return true;
}

async function main() {
    // Default source paths follow the GoReleaser v2 layout
    //   dist/<build id>_<os>_<arch>_<arch level>/<binary>
    // with build id `reflect`, binary `reflect` (Unix) / `reflect.exe` (Windows), and the
    // default arch levels GOAMD64=v1 and GOARM64=v8.0 — e.g.
    // dist/reflect_linux_amd64_v1/reflect and dist/reflect_darwin_arm64_v8.0/reflect. If the
    // config changes these, set the REFLECT_* env vars instead.
    const goosGoarch = {
        darwin: { arm64: 'arm64', x64: 'amd64' },
        linux: { arm64: 'arm64', x64: 'amd64' },
        win32: { arm64: 'arm64', x64: 'amd64' },
    };

    function gorel(platform, arch) {
        const ga = goosGoarch[platform]?.[arch];
        if (!ga) throw new Error(`no goarch mapping for ${platform}-${arch}`);
        const os = platform === 'win32' ? 'windows' : platform;
        const bin = platform === 'win32' ? 'reflect.exe' : 'reflect';
        const dirName = `reflect_${os}_${ga}_${ga === 'arm64' ? 'v8.0' : 'v1'}`;
        return path.join(distRoot, dirName, bin);
    }

    // All six targets are built by `.goreleaser.yml` (linux/darwin/windows ×
    // amd64/arm64).
    const targets = [
        {
            platform: 'darwin', arch: 'arm64', binName: 'reflect',
            envVar: 'REFLECT_DARWIN_ARM64',
            defaultSource: gorel('darwin', 'arm64'),
        },
        {
            platform: 'darwin', arch: 'x64', binName: 'reflect',
            envVar: 'REFLECT_DARWIN_X64',
            defaultSource: gorel('darwin', 'x64'),
        },
        {
            platform: 'linux', arch: 'x64', binName: 'reflect',
            envVar: 'REFLECT_LINUX_X64',
            defaultSource: gorel('linux', 'x64'),
        },
        {
            platform: 'linux', arch: 'arm64', binName: 'reflect',
            envVar: 'REFLECT_LINUX_ARM64',
            defaultSource: gorel('linux', 'arm64'),
        },
        {
            platform: 'win32', arch: 'x64', binName: 'reflect.exe',
            envVar: 'REFLECT_WIN32_X64',
            defaultSource: gorel('win32', 'x64'),
        },
        {
            platform: 'win32', arch: 'arm64', binName: 'reflect.exe',
            envVar: 'REFLECT_WIN32_ARM64',
            defaultSource: gorel('win32', 'arm64'),
        },
    ];

    // Compress in parallel — brotliCompress runs on the libuv thread pool so
    // calls genuinely overlap (set UV_THREADPOOL_SIZE>=6 in CI for full
    // parallelism; Node's default pool size is 4).
    const results = await Promise.all(targets.map(packPlatform));
    const failed = results.filter(r => !r).length;
    if (failed > 0) {
        console.error(`[assemble] ${failed} target(s) failed.`);
        process.exit(1);
    }

    // Pin the meta package and its optionalDependencies to the same version;
    // otherwise npm could resolve a different platform-package version.
    const meta = JSON.parse(fs.readFileSync(META_PKG_JSON, 'utf8'));
    meta.version = VERSION;
    for (const name of Object.keys(meta.optionalDependencies || {})) {
        meta.optionalDependencies[name] = VERSION;
    }
    fs.writeFileSync(META_PKG_JSON, JSON.stringify(meta, null, 4) + '\n');

    console.log(`[assemble] All ${targets.length} per-platform packages and the meta package assembled at version ${VERSION}.`);
}

main().catch((err) => { console.error(err); process.exit(1); });
