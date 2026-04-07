
// this file is generated — do not edit it


declare module "svelte/elements" {
	export interface HTMLAttributes<T> {
		'data-sveltekit-keepfocus'?: true | '' | 'off' | undefined | null;
		'data-sveltekit-noscroll'?: true | '' | 'off' | undefined | null;
		'data-sveltekit-preload-code'?:
			| true
			| ''
			| 'eager'
			| 'viewport'
			| 'hover'
			| 'tap'
			| 'off'
			| undefined
			| null;
		'data-sveltekit-preload-data'?: true | '' | 'hover' | 'tap' | 'off' | undefined | null;
		'data-sveltekit-reload'?: true | '' | 'off' | undefined | null;
		'data-sveltekit-replacestate'?: true | '' | 'off' | undefined | null;
	}
}

export {};


declare module "$app/types" {
	export interface AppTypes {
		RouteId(): "/" | "/admin" | "/ai_live_audio" | "/ai_live_audio/[lang]" | "/login" | "/subtitles";
		RouteParams(): {
			"/ai_live_audio/[lang]": { lang: string }
		};
		LayoutParams(): {
			"/": { lang?: string };
			"/admin": Record<string, never>;
			"/ai_live_audio": { lang?: string };
			"/ai_live_audio/[lang]": { lang: string };
			"/login": Record<string, never>;
			"/subtitles": Record<string, never>
		};
		Pathname(): "/" | "/admin" | `/ai_live_audio/${string}` & {} | "/login";
		ResolvedPathname(): `${"" | `/${string}`}${ReturnType<AppTypes['Pathname']>}`;
		Asset(): "/favicon.png" | string & {};
	}
}