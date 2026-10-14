import { chmod, copyFile, mkdir, rm, writeFile } from "node:fs/promises"
import { join, resolve } from "node:path"
import { parseArgs } from "node:util"

const TARGETS = [
	{ key: "darwin-arm64", os: "darwin", cpu: "arm64", binary: "cortex-darwin-arm64" },
	{ key: "darwin-x64", os: "darwin", cpu: "x64", binary: "cortex-darwin-amd64" },
	{ key: "linux-x64", os: "linux", cpu: "x64", binary: "cortex-linux-amd64" },
	{ key: "linux-arm64", os: "linux", cpu: "arm64", binary: "cortex-linux-arm64" },
	{ key: "win32-x64", os: "win32", cpu: "x64", binary: "cortex-windows-amd64.exe" },
]

const SHARED = {
	license: "MIT",
	repository: { type: "git", url: "https://github.com/cortex-md/cli" },
	homepage: "https://github.com/cortex-md/cli#readme",
}

const { values } = parseArgs({
	options: {
		version: { type: "string" },
		bin: { type: "string", default: "bin" },
		out: { type: "string", default: "dist/npm" },
		only: { type: "string", multiple: true },
	},
})

if (!values.version || !/^\d+\.\d+\.\d+(-[0-9A-Za-z.-]+)?$/.test(values.version)) {
	throw new Error("--version x.y.z is required")
}

const version = values.version
const binDir = resolve(values.bin)
const outDir = resolve(values.out)
const targets = values.only ? TARGETS.filter((target) => values.only.includes(target.key)) : TARGETS

await rm(outDir, { recursive: true, force: true })

for (const target of targets) {
	const dir = join(outDir, `cli-${target.key}`)
	const binaryName = target.os === "win32" ? "cortex.exe" : "cortex"
	await mkdir(join(dir, "bin"), { recursive: true })
	await copyFile(join(binDir, target.binary), join(dir, "bin", binaryName))
	await chmod(join(dir, "bin", binaryName), 0o755)
	await writeJson(join(dir, "package.json"), {
		name: `@cortex.md/cli-${target.key}`,
		version,
		description: `Cortex CLI binary for ${target.os} ${target.cpu}`,
		...SHARED,
		os: [target.os],
		cpu: [target.cpu],
		files: ["bin"],
		preferUnplugged: true,
	})
}

const mainDir = join(outDir, "cli")
await mkdir(join(mainDir, "bin"), { recursive: true })
await copyFile(resolve("npm/cortex.js"), join(mainDir, "bin", "cortex.js"))
await chmod(join(mainDir, "bin", "cortex.js"), 0o755)
await copyFile(resolve("README.md"), join(mainDir, "README.md"))
await writeJson(join(mainDir, "package.json"), {
	name: "@cortex.md/cli",
	version,
	description: "Build, test and publish Cortex plugins and themes",
	...SHARED,
	keywords: ["cortex", "cli", "plugin", "theme"],
	bin: { cortex: "bin/cortex.js" },
	files: ["bin", "README.md"],
	engines: { node: ">=18" },
	optionalDependencies: Object.fromEntries(TARGETS.map((target) => [`@cortex.md/cli-${target.key}`, version])),
})

console.log(`npm packages for ${version} written to ${outDir}`)

async function writeJson(path, value) {
	await writeFile(path, `${JSON.stringify(value, null, "\t")}\n`)
}
