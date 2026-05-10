
// this file is generated — do not edit it


/// <reference types="@sveltejs/kit" />

/**
 * This module provides access to environment variables that are injected _statically_ into your bundle at build time and are limited to _private_ access.
 * 
 * |         | Runtime                                                                    | Build time                                                               |
 * | ------- | -------------------------------------------------------------------------- | ------------------------------------------------------------------------ |
 * | Private | [`$env/dynamic/private`](https://svelte.dev/docs/kit/$env-dynamic-private) | [`$env/static/private`](https://svelte.dev/docs/kit/$env-static-private) |
 * | Public  | [`$env/dynamic/public`](https://svelte.dev/docs/kit/$env-dynamic-public)   | [`$env/static/public`](https://svelte.dev/docs/kit/$env-static-public)   |
 * 
 * Static environment variables are [loaded by Vite](https://vitejs.dev/guide/env-and-mode.html#env-files) from `.env` files and `process.env` at build time and then statically injected into your bundle at build time, enabling optimisations like dead code elimination.
 * 
 * **_Private_ access:**
 * 
 * - This module cannot be imported into client-side code
 * - This module only includes variables that _do not_ begin with [`config.kit.env.publicPrefix`](https://svelte.dev/docs/kit/configuration#env) _and do_ start with [`config.kit.env.privatePrefix`](https://svelte.dev/docs/kit/configuration#env) (if configured)
 * 
 * For example, given the following build time environment:
 * 
 * ```env
 * ENVIRONMENT=production
 * PUBLIC_BASE_URL=http://site.com
 * ```
 * 
 * With the default `publicPrefix` and `privatePrefix`:
 * 
 * ```ts
 * import { ENVIRONMENT, PUBLIC_BASE_URL } from '$env/static/private';
 * 
 * console.log(ENVIRONMENT); // => "production"
 * console.log(PUBLIC_BASE_URL); // => throws error during build
 * ```
 * 
 * The above values will be the same _even if_ different values for `ENVIRONMENT` or `PUBLIC_BASE_URL` are set at runtime, as they are statically replaced in your code with their build time values.
 */
declare module '$env/static/private' {
	export const GJS_DEBUG_TOPICS: string;
	export const VSCODE_CWD: string;
	export const LANGUAGE: string;
	export const USER: string;
	export const VSCODE_NLS_CONFIG: string;
	export const npm_config_user_agent: string;
	export const XDG_SESSION_TYPE: string;
	export const __GLX_VENDOR_LIBRARY_NAME: string;
	export const npm_node_execpath: string;
	export const SHLVL: string;
	export const npm_config_noproxy: string;
	export const LESS: string;
	export const HOME: string;
	export const CHROME_DESKTOP: string;
	export const OLDPWD: string;
	export const DESKTOP_SESSION: string;
	export const NVM_BIN: string;
	export const npm_package_json: string;
	export const LSCOLORS: string;
	export const NVM_INC: string;
	export const QT_STYLE_OVERRIDE: string;
	export const VSCODE_IPC_HOOK: string;
	export const ZSH: string;
	export const GIO_LAUNCHED_DESKTOP_FILE: string;
	export const GNOME_SHELL_SESSION_MODE: string;
	export const TICKTICK_API_KEY: string;
	export const GTK_MODULES: string;
	export const PAGER: string;
	export const MANAGERPID: string;
	export const SUDO_ASKPASS: string;
	export const npm_config_userconfig: string;
	export const npm_config_local_prefix: string;
	export const SYSTEMD_EXEC_PID: string;
	export const SSH_ASKPASS: string;
	export const DBUS_SESSION_BUS_ADDRESS: string;
	export const TICKTICK_PASSWORD: string;
	export const ANTIGRAVITY_AGENT: string;
	export const GIO_LAUNCHED_DESKTOP_FILE_PID: string;
	export const COLOR: string;
	export const NVM_DIR: string;
	export const WAYLAND_DISPLAY: string;
	export const LOGNAME: string;
	export const __VK_LAYER_NV_optimus: string;
	export const _: string;
	export const JOURNAL_STREAM: string;
	export const npm_config_prefix: string;
	export const npm_config_npm_version: string;
	export const MEMORY_PRESSURE_WATCH: string;
	export const XDG_SESSION_CLASS: string;
	export const USERNAME: string;
	export const TERM: string;
	export const TICKTICK_EMAIL: string;
	export const npm_config_cache: string;
	export const GNOME_DESKTOP_SESSION_ID: string;
	export const FC_FONTATIONS: string;
	export const ANTIGRAVITY_EDITOR_APP_ROOT: string;
	export const npm_config_node_gyp: string;
	export const PATH: string;
	export const INVOCATION_ID: string;
	export const NODE: string;
	export const npm_package_name: string;
	export const XDG_MENU_PREFIX: string;
	export const GNOME_SETUP_DISPLAY: string;
	export const XDG_RUNTIME_DIR: string;
	export const GDK_BACKEND: string;
	export const GBM_BACKEND: string;
	export const DISPLAY: string;
	export const LANG: string;
	export const XDG_CURRENT_DESKTOP: string;
	export const __NV_PRIME_RENDER_OFFLOAD: string;
	export const XMODIFIERS: string;
	export const XDG_SESSION_DESKTOP: string;
	export const XAUTHORITY: string;
	export const LS_COLORS: string;
	export const SBX_CHROME_API_RQ: string;
	export const npm_lifecycle_script: string;
	export const SSH_AUTH_SOCK: string;
	export const SHELL: string;
	export const npm_package_version: string;
	export const npm_lifecycle_event: string;
	export const QT_ACCESSIBILITY: string;
	export const GDMSESSION: string;
	export const GPG_AGENT_INFO: string;
	export const GJS_DEBUG_OUTPUT: string;
	export const Z3_LIBRARY_PATH: string;
	export const QT_IM_MODULE: string;
	export const ANTIGRAVITY_TRAJECTORY_ID: string;
	export const npm_config_globalconfig: string;
	export const npm_config_init_module: string;
	export const JAVA_HOME: string;
	export const PWD: string;
	export const FZF_DEFAULT_COMMAND: string;
	export const LC_ALL: string;
	export const npm_execpath: string;
	export const VSCODE_CODE_CACHE_PATH: string;
	export const NVM_CD_FLAGS: string;
	export const XDG_DATA_DIRS: string;
	export const npm_config_global_prefix: string;
	export const QTWEBENGINE_DICTIONARIES_PATH: string;
	export const TICKTICK_API_SECRET: string;
	export const npm_command: string;
	export const QT_IM_MODULES: string;
	export const MEMORY_PRESSURE_WRITE: string;
	export const EDITOR: string;
	export const VSCODE_PID: string;
	export const INIT_CWD: string;
	export const TEST: string;
	export const VITEST: string;
	export const NODE_ENV: string;
	export const PROD: string;
	export const DEV: string;
	export const BASE_URL: string;
	export const MODE: string;
}

/**
 * This module provides access to environment variables that are injected _statically_ into your bundle at build time and are _publicly_ accessible.
 * 
 * |         | Runtime                                                                    | Build time                                                               |
 * | ------- | -------------------------------------------------------------------------- | ------------------------------------------------------------------------ |
 * | Private | [`$env/dynamic/private`](https://svelte.dev/docs/kit/$env-dynamic-private) | [`$env/static/private`](https://svelte.dev/docs/kit/$env-static-private) |
 * | Public  | [`$env/dynamic/public`](https://svelte.dev/docs/kit/$env-dynamic-public)   | [`$env/static/public`](https://svelte.dev/docs/kit/$env-static-public)   |
 * 
 * Static environment variables are [loaded by Vite](https://vitejs.dev/guide/env-and-mode.html#env-files) from `.env` files and `process.env` at build time and then statically injected into your bundle at build time, enabling optimisations like dead code elimination.
 * 
 * **_Public_ access:**
 * 
 * - This module _can_ be imported into client-side code
 * - **Only** variables that begin with [`config.kit.env.publicPrefix`](https://svelte.dev/docs/kit/configuration#env) (which defaults to `PUBLIC_`) are included
 * 
 * For example, given the following build time environment:
 * 
 * ```env
 * ENVIRONMENT=production
 * PUBLIC_BASE_URL=http://site.com
 * ```
 * 
 * With the default `publicPrefix` and `privatePrefix`:
 * 
 * ```ts
 * import { ENVIRONMENT, PUBLIC_BASE_URL } from '$env/static/public';
 * 
 * console.log(ENVIRONMENT); // => throws error during build
 * console.log(PUBLIC_BASE_URL); // => "http://site.com"
 * ```
 * 
 * The above values will be the same _even if_ different values for `ENVIRONMENT` or `PUBLIC_BASE_URL` are set at runtime, as they are statically replaced in your code with their build time values.
 */
declare module '$env/static/public' {
	
}

/**
 * This module provides access to environment variables set _dynamically_ at runtime and that are limited to _private_ access.
 * 
 * |         | Runtime                                                                    | Build time                                                               |
 * | ------- | -------------------------------------------------------------------------- | ------------------------------------------------------------------------ |
 * | Private | [`$env/dynamic/private`](https://svelte.dev/docs/kit/$env-dynamic-private) | [`$env/static/private`](https://svelte.dev/docs/kit/$env-static-private) |
 * | Public  | [`$env/dynamic/public`](https://svelte.dev/docs/kit/$env-dynamic-public)   | [`$env/static/public`](https://svelte.dev/docs/kit/$env-static-public)   |
 * 
 * Dynamic environment variables are defined by the platform you're running on. For example if you're using [`adapter-node`](https://github.com/sveltejs/kit/tree/main/packages/adapter-node) (or running [`vite preview`](https://svelte.dev/docs/kit/cli)), this is equivalent to `process.env`.
 * 
 * **_Private_ access:**
 * 
 * - This module cannot be imported into client-side code
 * - This module includes variables that _do not_ begin with [`config.kit.env.publicPrefix`](https://svelte.dev/docs/kit/configuration#env) _and do_ start with [`config.kit.env.privatePrefix`](https://svelte.dev/docs/kit/configuration#env) (if configured)
 * 
 * > [!NOTE] In `dev`, `$env/dynamic` includes environment variables from `.env`. In `prod`, this behavior will depend on your adapter.
 * 
 * > [!NOTE] To get correct types, environment variables referenced in your code should be declared (for example in an `.env` file), even if they don't have a value until the app is deployed:
 * >
 * > ```env
 * > MY_FEATURE_FLAG=
 * > ```
 * >
 * > You can override `.env` values from the command line like so:
 * >
 * > ```sh
 * > MY_FEATURE_FLAG="enabled" npm run dev
 * > ```
 * 
 * For example, given the following runtime environment:
 * 
 * ```env
 * ENVIRONMENT=production
 * PUBLIC_BASE_URL=http://site.com
 * ```
 * 
 * With the default `publicPrefix` and `privatePrefix`:
 * 
 * ```ts
 * import { env } from '$env/dynamic/private';
 * 
 * console.log(env.ENVIRONMENT); // => "production"
 * console.log(env.PUBLIC_BASE_URL); // => undefined
 * ```
 */
declare module '$env/dynamic/private' {
	export const env: {
		GJS_DEBUG_TOPICS: string;
		VSCODE_CWD: string;
		LANGUAGE: string;
		USER: string;
		VSCODE_NLS_CONFIG: string;
		npm_config_user_agent: string;
		XDG_SESSION_TYPE: string;
		__GLX_VENDOR_LIBRARY_NAME: string;
		npm_node_execpath: string;
		SHLVL: string;
		npm_config_noproxy: string;
		LESS: string;
		HOME: string;
		CHROME_DESKTOP: string;
		OLDPWD: string;
		DESKTOP_SESSION: string;
		NVM_BIN: string;
		npm_package_json: string;
		LSCOLORS: string;
		NVM_INC: string;
		QT_STYLE_OVERRIDE: string;
		VSCODE_IPC_HOOK: string;
		ZSH: string;
		GIO_LAUNCHED_DESKTOP_FILE: string;
		GNOME_SHELL_SESSION_MODE: string;
		TICKTICK_API_KEY: string;
		GTK_MODULES: string;
		PAGER: string;
		MANAGERPID: string;
		SUDO_ASKPASS: string;
		npm_config_userconfig: string;
		npm_config_local_prefix: string;
		SYSTEMD_EXEC_PID: string;
		SSH_ASKPASS: string;
		DBUS_SESSION_BUS_ADDRESS: string;
		TICKTICK_PASSWORD: string;
		ANTIGRAVITY_AGENT: string;
		GIO_LAUNCHED_DESKTOP_FILE_PID: string;
		COLOR: string;
		NVM_DIR: string;
		WAYLAND_DISPLAY: string;
		LOGNAME: string;
		__VK_LAYER_NV_optimus: string;
		_: string;
		JOURNAL_STREAM: string;
		npm_config_prefix: string;
		npm_config_npm_version: string;
		MEMORY_PRESSURE_WATCH: string;
		XDG_SESSION_CLASS: string;
		USERNAME: string;
		TERM: string;
		TICKTICK_EMAIL: string;
		npm_config_cache: string;
		GNOME_DESKTOP_SESSION_ID: string;
		FC_FONTATIONS: string;
		ANTIGRAVITY_EDITOR_APP_ROOT: string;
		npm_config_node_gyp: string;
		PATH: string;
		INVOCATION_ID: string;
		NODE: string;
		npm_package_name: string;
		XDG_MENU_PREFIX: string;
		GNOME_SETUP_DISPLAY: string;
		XDG_RUNTIME_DIR: string;
		GDK_BACKEND: string;
		GBM_BACKEND: string;
		DISPLAY: string;
		LANG: string;
		XDG_CURRENT_DESKTOP: string;
		__NV_PRIME_RENDER_OFFLOAD: string;
		XMODIFIERS: string;
		XDG_SESSION_DESKTOP: string;
		XAUTHORITY: string;
		LS_COLORS: string;
		SBX_CHROME_API_RQ: string;
		npm_lifecycle_script: string;
		SSH_AUTH_SOCK: string;
		SHELL: string;
		npm_package_version: string;
		npm_lifecycle_event: string;
		QT_ACCESSIBILITY: string;
		GDMSESSION: string;
		GPG_AGENT_INFO: string;
		GJS_DEBUG_OUTPUT: string;
		Z3_LIBRARY_PATH: string;
		QT_IM_MODULE: string;
		ANTIGRAVITY_TRAJECTORY_ID: string;
		npm_config_globalconfig: string;
		npm_config_init_module: string;
		JAVA_HOME: string;
		PWD: string;
		FZF_DEFAULT_COMMAND: string;
		LC_ALL: string;
		npm_execpath: string;
		VSCODE_CODE_CACHE_PATH: string;
		NVM_CD_FLAGS: string;
		XDG_DATA_DIRS: string;
		npm_config_global_prefix: string;
		QTWEBENGINE_DICTIONARIES_PATH: string;
		TICKTICK_API_SECRET: string;
		npm_command: string;
		QT_IM_MODULES: string;
		MEMORY_PRESSURE_WRITE: string;
		EDITOR: string;
		VSCODE_PID: string;
		INIT_CWD: string;
		TEST: string;
		VITEST: string;
		NODE_ENV: string;
		PROD: string;
		DEV: string;
		BASE_URL: string;
		MODE: string;
		[key: `PUBLIC_${string}`]: undefined;
		[key: `${string}`]: string | undefined;
	}
}

/**
 * This module provides access to environment variables set _dynamically_ at runtime and that are _publicly_ accessible.
 * 
 * |         | Runtime                                                                    | Build time                                                               |
 * | ------- | -------------------------------------------------------------------------- | ------------------------------------------------------------------------ |
 * | Private | [`$env/dynamic/private`](https://svelte.dev/docs/kit/$env-dynamic-private) | [`$env/static/private`](https://svelte.dev/docs/kit/$env-static-private) |
 * | Public  | [`$env/dynamic/public`](https://svelte.dev/docs/kit/$env-dynamic-public)   | [`$env/static/public`](https://svelte.dev/docs/kit/$env-static-public)   |
 * 
 * Dynamic environment variables are defined by the platform you're running on. For example if you're using [`adapter-node`](https://github.com/sveltejs/kit/tree/main/packages/adapter-node) (or running [`vite preview`](https://svelte.dev/docs/kit/cli)), this is equivalent to `process.env`.
 * 
 * **_Public_ access:**
 * 
 * - This module _can_ be imported into client-side code
 * - **Only** variables that begin with [`config.kit.env.publicPrefix`](https://svelte.dev/docs/kit/configuration#env) (which defaults to `PUBLIC_`) are included
 * 
 * > [!NOTE] In `dev`, `$env/dynamic` includes environment variables from `.env`. In `prod`, this behavior will depend on your adapter.
 * 
 * > [!NOTE] To get correct types, environment variables referenced in your code should be declared (for example in an `.env` file), even if they don't have a value until the app is deployed:
 * >
 * > ```env
 * > MY_FEATURE_FLAG=
 * > ```
 * >
 * > You can override `.env` values from the command line like so:
 * >
 * > ```sh
 * > MY_FEATURE_FLAG="enabled" npm run dev
 * > ```
 * 
 * For example, given the following runtime environment:
 * 
 * ```env
 * ENVIRONMENT=production
 * PUBLIC_BASE_URL=http://example.com
 * ```
 * 
 * With the default `publicPrefix` and `privatePrefix`:
 * 
 * ```ts
 * import { env } from '$env/dynamic/public';
 * console.log(env.ENVIRONMENT); // => undefined, not public
 * console.log(env.PUBLIC_BASE_URL); // => "http://example.com"
 * ```
 * 
 * ```
 * 
 * ```
 */
declare module '$env/dynamic/public' {
	export const env: {
		[key: `PUBLIC_${string}`]: string | undefined;
	}
}
