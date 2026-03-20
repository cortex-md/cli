import { CortexPlugin } from "cortex-plugin-api"

export default class {{CLASS_NAME}}Plugin extends CortexPlugin {
	onload(): void {
		this.addCommand({
			id: "example-command",
			label: "Example Command",
			icon: "smile",
			execute: () => {
				console.log("{{NAME}} is working!")
			},
		})

		this.registerStatusBarItem({
			id: "{{ID}}-status",
			position: "right",
			icon: "puzzle",
			text: "{{NAME}}",
			tooltip: "{{DESCRIPTION}}",
		})
	}

	onunload(): void {
		console.log("{{NAME}} unloaded")
	}
}
