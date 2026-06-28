<script lang="ts">
  import { getAppContext } from "$lib/audioState.svelte";
  let { 
    selected = '', 
    options = [], 
    onchange, 
    placeholder = 'Select language...',
    class: className = '' 
  } = $props();

  const { audio, ai } = getAppContext();
  const languages = [
    { code: 'en', name: 'English' },
    { code: 'nl', name: 'Dutch' },
    { code: 'pt', name: 'Portuguese' },
    { code: 'es', name: 'Spanish' },
    { code: 'fr', name: 'French' },
    { code: 'de', name: 'German' },
    { code: 'ru', name: 'Russian' },
    { code: 'tr', name: 'Turkish' },
    { code: 'pl', name: 'Polish' },
    { code: 'id', name: 'Indonesian' }
  ];

  const currentOptions = $derived(options.length > 0 ? options : (ai.aiConfig.languages.length > 0 ? ai.aiConfig.languages : languages));
</script>

<label for="lang-select" class="sr-only">{placeholder}</label>
<select 
  id="lang-select"
  class="bg-background border border-border text-foreground text-sm rounded-md focus:ring-primary focus:border-primary block w-full p-2.5 {className}"
  value={selected}
  onchange={(e) => onchange(e.currentTarget.value)}
>
  <option value="" disabled selected>{placeholder}</option>
  {#each currentOptions as lang}
    <option value={lang.code}>{lang.name}</option>
  {/each}
</select>
