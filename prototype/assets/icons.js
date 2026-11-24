const iconNames = [
  'dashboard', 'inbound', 'nodes', 'route', 'rss', 'rules', 'activity', 'settings',
  'search', 'bell', 'sun', 'moon', 'play', 'stop', 'plus', 'refresh',
  'chevron-up', 'chevron-down', 'chevron-right', 'arrow-up-right', 'check', 'alert', 'terminal',
  'lock', 'globe', 'cpu', 'clock', 'more', 'download', 'upload', 'drag', 'close',
  'menu', 'filter', 'link', 'shield', 'zap', 'database', 'sparkles', 'external',
  'layers', 'sliders', 'gauge', 'compass', 'radio', 'eye', 'eye-off', 'power', 'copy', 'trash',
];

const paths = {
  dashboard: '<rect x="3" y="3" width="7" height="9" rx="1.5"/><rect x="14" y="3" width="7" height="5" rx="1.5"/><rect x="14" y="12" width="7" height="9" rx="1.5"/><rect x="3" y="16" width="7" height="5" rx="1.5"/>',
  inbound: '<path d="M12 3v12"/><path d="m7 10 5 5 5-5"/><path d="M5 21h14"/>',
  nodes: '<rect x="3" y="3" width="7" height="7" rx="1.5"/><rect x="14" y="3" width="7" height="7" rx="1.5"/><rect x="3" y="14" width="7" height="7" rx="1.5"/><rect x="14" y="14" width="7" height="7" rx="1.5"/><path d="M10 6.5h4M6.5 10v4M17.5 10v4M10 17.5h4"/>',
  route: '<circle cx="6" cy="18" r="2"/><circle cx="18" cy="6" r="2"/><path d="M8 18h3a3 3 0 0 0 3-3V9a3 3 0 0 1 3-3h-1"/>',
  rss: '<path d="M4 11a9 9 0 0 1 9 9"/><path d="M4 4a16 16 0 0 1 16 16"/><circle cx="5" cy="19" r="1"/>',
  rules: '<path d="M9 5h11M9 12h11M9 19h11"/><path d="m3.5 5 1 1 2-2M3.5 12l1 1 2-2M3.5 19l1 1 2-2"/>',
  activity: '<path d="M3 12h4l2.5-7 5 14 2.5-7h4"/>',
  settings: '<circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.7 1.7 0 0 0 .34 1.88l.06.06a2 2 0 1 1-2.83 2.83l-.06-.06A1.7 1.7 0 0 0 15 19.4a1.7 1.7 0 0 0-1 .6 1.7 1.7 0 0 0-.4 1.1V21a2 2 0 1 1-4 0v-.09A1.7 1.7 0 0 0 8.6 19.4a1.7 1.7 0 0 0-1.88.34l-.06.06a2 2 0 1 1-2.83-2.83l.06-.06A1.7 1.7 0 0 0 4.6 15a1.7 1.7 0 0 0-.6-1 1.7 1.7 0 0 0-1.1-.4H3a2 2 0 1 1 0-4h.09A1.7 1.7 0 0 0 4.6 8.6a1.7 1.7 0 0 0-.34-1.88l-.06-.06a2 2 0 1 1 2.83-2.83l.06.06A1.7 1.7 0 0 0 9 4.6a1.7 1.7 0 0 0 1-.6 1.7 1.7 0 0 0 .4-1.1V3a2 2 0 1 1 4 0v.09A1.7 1.7 0 0 0 15.4 4.6a1.7 1.7 0 0 0 1.88-.34l.06-.06a2 2 0 1 1 2.83 2.83l-.06.06A1.7 1.7 0 0 0 19.4 9c.1.38.32.72.6 1 .3.26.68.4 1.1.4H21a2 2 0 1 1 0 4h-.09a1.7 1.7 0 0 0-1.51.6Z"/>',
  search: '<circle cx="11" cy="11" r="7"/><path d="m20 20-4-4"/>',
  bell: '<path d="M18 8a6 6 0 0 0-12 0c0 7-3 7-3 9h18c0-2-3-2-3-9M10 21h4"/>',
  sun: '<circle cx="12" cy="12" r="4"/><path d="M12 2v2M12 20v2M4.93 4.93l1.42 1.42M17.66 17.66l1.41 1.41M2 12h2M20 12h2M4.93 19.07l1.42-1.42M17.66 6.34l1.41-1.41"/>',
  moon: '<path d="M20.5 14.5A8.5 8.5 0 0 1 9.5 3.5 8.5 8.5 0 1 0 20.5 14.5Z"/>',
  play: '<path d="m8 5 11 7-11 7V5Z"/>',
  stop: '<rect x="5" y="5" width="14" height="14" rx="2"/>',
  plus: '<path d="M12 5v14M5 12h14"/>',
  refresh: '<path d="M20 7v5h-5M4 17v-5h5"/><path d="M6.1 9A7 7 0 0 1 18.7 7M17.9 15A7 7 0 0 1 5.3 17"/>',
  'chevron-up': '<path d="m6 15 6-6 6 6"/>',
  'chevron-down': '<path d="m6 9 6 6 6-6"/>',
  'chevron-right': '<path d="m9 18 6-6-6-6"/>',
  'arrow-up-right': '<path d="M7 17 17 7M8 7h9v9"/>',
  check: '<path d="m5 12 4 4L19 6"/>',
  alert: '<path d="M12 3 2.8 20h18.4L12 3Z"/><path d="M12 9v5M12 17.5v.1"/>',
  terminal: '<rect x="3" y="4" width="18" height="16" rx="2"/><path d="m7 9 3 3-3 3M13 15h4"/>',
  lock: '<rect x="4" y="10" width="16" height="11" rx="2"/><path d="M8 10V7a4 4 0 0 1 8 0v3"/>',
  globe: '<circle cx="12" cy="12" r="9"/><path d="M3 12h18M12 3a15 15 0 0 1 0 18M12 3a15 15 0 0 0 0 18"/>',
  cpu: '<rect x="7" y="7" width="10" height="10" rx="2"/><path d="M9 2v3M15 2v3M9 19v3M15 19v3M2 9h3M2 15h3M19 9h3M19 15h3"/><rect x="10" y="10" width="4" height="4" rx="1"/>',
  clock: '<circle cx="12" cy="12" r="9"/><path d="M12 7v5l3 2"/>',
  more: '<circle cx="5" cy="12" r="1"/><circle cx="12" cy="12" r="1"/><circle cx="19" cy="12" r="1"/>',
  download: '<path d="M12 3v12M7 10l5 5 5-5M4 21h16"/>',
  upload: '<path d="M12 15V3M7 8l5-5 5 5M4 21h16"/>',
  drag: '<circle cx="9" cy="6" r="1"/><circle cx="15" cy="6" r="1"/><circle cx="9" cy="12" r="1"/><circle cx="15" cy="12" r="1"/><circle cx="9" cy="18" r="1"/><circle cx="15" cy="18" r="1"/>',
  close: '<path d="M6 6l12 12M18 6 6 18"/>',
  menu: '<path d="M4 7h16M4 12h16M4 17h16"/>',
  filter: '<path d="M4 5h16l-6 7v6l-4 2v-8L4 5Z"/>',
  link: '<path d="M10 13a5 5 0 0 0 7.5.5l2-2a5 5 0 0 0-7-7l-1.2 1.2M14 11a5 5 0 0 0-7.5-.5l-2 2a5 5 0 0 0 7 7l1.2-1.2"/>',
  shield: '<path d="M12 3 4.5 6v5.5c0 4.6 3.2 7.8 7.5 9.5 4.3-1.7 7.5-4.9 7.5-9.5V6L12 3Z"/><path d="m9 12 2 2 4-4"/>',
  zap: '<path d="M13 2 4 14h6l-1 8 9-12h-6l1-8Z"/>',
  database: '<ellipse cx="12" cy="5" rx="8" ry="3"/><path d="M4 5v7c0 1.7 3.6 3 8 3s8-1.3 8-3V5M4 12v7c0 1.7 3.6 3 8 3s8-1.3 8-3v-7"/>',
  sparkles: '<path d="m12 3 1.3 4.2L17.5 8.5l-4.2 1.3L12 14l-1.3-4.2-4.2-1.3 4.2-1.3L12 3ZM19 14l.8 2.2L22 17l-2.2.8L19 20l-.8-2.2L16 17l2.2-.8L19 14ZM5 14l.6 1.4L7 16l-1.4.6L5 18l-.6-1.4L3 16l1.4-.6L5 14Z"/>',
  external: '<path d="M14 3h7v7M10 14 21 3M21 14v5a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5"/>',
  layers: '<path d="m12 3 9 5-9 5-9-5 9-5Z"/><path d="m3 12 9 5 9-5M3 16l9 5 9-5"/>',
  sliders: '<path d="M4 5h4M12 5h8M4 12h10M18 12h2M4 19h2M10 19h10"/><circle cx="10" cy="5" r="2"/><circle cx="16" cy="12" r="2"/><circle cx="8" cy="19" r="2"/>',
  gauge: '<path d="M4.9 19a9 9 0 1 1 14.2 0"/><path d="m12 14 4-4"/><path d="M5 19h14"/>',
  compass: '<circle cx="12" cy="12" r="9"/><path d="m16 8-2.5 5.5L8 16l2.5-5.5L16 8Z"/>',
  radio: '<circle cx="12" cy="12" r="2"/><path d="M7.8 7.8a6 6 0 0 0 0 8.4M16.2 7.8a6 6 0 0 1 0 8.4M4.9 4.9a10 10 0 0 0 0 14.2M19.1 4.9a10 10 0 0 1 0 14.2"/>',
  eye: '<path d="M2 12s3.5-6 10-6 10 6 10 6-3.5 6-10 6S2 12 2 12Z"/><circle cx="12" cy="12" r="2.5"/>',
  'eye-off': '<path d="m3 3 18 18M10.6 10.7a2 2 0 0 0 2.7 2.7M9.9 5.2A10.5 10.5 0 0 1 12 5c6.5 0 10 7 10 7a17 17 0 0 1-2.1 3.2M6.6 6.6C3.6 8.5 2 12 2 12s3.5 7 10 7a10 10 0 0 0 4.1-.9"/>',
  power: '<path d="M12 2v9"/><path d="M6.4 5.4a9 9 0 1 0 11.2 0"/>',
  copy: '<rect x="8" y="8" width="12" height="12" rx="2"/><path d="M16 8V5a1 1 0 0 0-1-1H5a1 1 0 0 0-1 1v10a1 1 0 0 0 1 1h3"/>',
  trash: '<path d="M4 7h16M9 7V5a1 1 0 0 1 1-1h4a1 1 0 0 1 1 1v2M7 7l1 12a2 2 0 0 0 2 2h4a2 2 0 0 0 2-2l1-12"/><path d="M10 11v6M14 11v6"/>',
};

export function icon(name, className = '') {
  const safeName = iconNames.includes(name) ? name : 'dashboard';
  return `<svg class="icon ${className}" aria-hidden="true"><use href="#i-${safeName}"></use></svg>`;
}

export function iconSprite() {
  const symbols = iconNames
    .map((name) => `<symbol id="i-${name}" viewBox="0 0 24 24">${paths[name]}</symbol>`)
    .join('');
  return `<svg class="icon-sprite" aria-hidden="true" focusable="false">${symbols}</svg>`;
}

export const ICON_NAMES = iconNames;
