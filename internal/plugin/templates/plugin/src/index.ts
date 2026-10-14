import { definePlugin, type PluginManifest } from "@cortex.md/api"
import manifest from "../manifest.json"

export default definePlugin({
	manifest: manifest as PluginManifest,
	contributes: {
		commands: [
			{
				id: "say-hello",
				label: "{{NAME}}: Say hello",
				icon: "sparkles",
				run: (ctx) => ctx.ui.showNotice("Hello from {{NAME}}"),
			},
		],
	},
})
