const fs = require("node:fs")

const [bundlePath, permissionsJSON] = process.argv.slice(2)
const result = { manifest: null, hasMarkdownContributions: false, error: null }

const stub = new Proxy(function () {}, {
	get: (_target, property) => (property === "then" ? undefined : stub),
	apply: () => stub,
	construct: () => stub,
})

const api = {
	definePlugin: (definition) => definition,
	PLUGIN_PERMISSIONS: JSON.parse(permissionsJSON),
}

function requireExternal(id) {
	if (id === "@cortex.md/api") return api
	if (id === "@codemirror/state" || id === "@codemirror/view") return stub
	throw new Error(`Cannot require "${id}"`)
}

try {
	const code = fs.readFileSync(bundlePath, "utf8")
	const moduleObject = { exports: {} }
	const factory = (0, eval)(`(function(module, exports, require) {\n${code}\n})`)
	factory(moduleObject, moduleObject.exports, requireExternal)
	const definition = moduleObject.exports.default ?? moduleObject.exports
	if (!definition || typeof definition !== "object" || !definition.manifest) {
		throw new Error("Plugin bundle must export a definePlugin(...) object as default")
	}
	result.manifest = definition.manifest
	result.hasMarkdownContributions = Boolean(definition.contributes && definition.contributes.markdown)
} catch (error) {
	result.error = error instanceof Error ? error.message : String(error)
}

process.stdout.write(JSON.stringify(result))
