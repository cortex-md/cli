#!/usr/bin/env node
const { spawn } = require("node:child_process")

const platformKey = `${process.platform}-${process.arch}`
const packageName = `@cortex.md/cli-${platformKey}`
const binaryName = process.platform === "win32" ? "cortex.exe" : "cortex"

let binary
try {
	binary = require.resolve(`${packageName}/bin/${binaryName}`)
} catch {
	console.error(
		`Cortex CLI has no binary for ${platformKey}, or the optional package ${packageName} was not installed.\n` +
			"Reinstall without --no-optional / --omit=optional, or download a binary from https://github.com/cortex-md/cli/releases",
	)
	process.exit(1)
}

const child = spawn(binary, process.argv.slice(2), { stdio: "inherit" })

process.on("SIGINT", () => {})
process.on("SIGTERM", () => child.kill("SIGTERM"))

child.on("error", (error) => {
	console.error(error.message)
	process.exit(1)
})

child.on("exit", (code, signal) => {
	process.exit(code ?? (signal ? 1 : 0))
})
