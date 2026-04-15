const BASE = [
  { author: 'nova', avatarColor: '#ff3864', gradient: 'linear-gradient(135deg,#ff3864,#7a1fff)', title: 'Morning light', caption: 'Caught the sunrise over the rooftops this morning. Felt like a whole new city.', likes: 128, comments: 14 },
  { author: 'kaito', avatarColor: '#38bdf8', gradient: 'linear-gradient(135deg,#0ea5e9,#6366f1)', title: 'Neon alley', caption: 'Late night wandering. The neon signs make every puddle a painting.', likes: 402, comments: 37 },
  { author: 'mira', avatarColor: '#f59e0b', gradient: 'linear-gradient(135deg,#f59e0b,#ef4444)', title: 'Desert dust', caption: 'Two days in the dunes. No signal, no noise. Just wind and sand.', likes: 89, comments: 6 },
  { author: 'ryo', avatarColor: '#22c55e', gradient: 'linear-gradient(135deg,#22c55e,#14b8a6)', title: 'Forest frame', caption: 'Moss everywhere. The forest eats sound. Best kind of silence.', likes: 211, comments: 19 },
  { author: 'lumi', avatarColor: '#a855f7', gradient: 'linear-gradient(135deg,#a855f7,#ec4899)', title: 'Glass city', caption: 'Reflections on reflections. The skyline bends in every window.', likes: 567, comments: 58 },
  { author: 'tessa', avatarColor: '#eab308', gradient: 'linear-gradient(135deg,#eab308,#f97316)', title: 'Golden hour', caption: 'Ten minutes of perfect light. Worth the three hour wait.', likes: 344, comments: 22 },
  { author: 'ivo', avatarColor: '#06b6d4', gradient: 'linear-gradient(135deg,#06b6d4,#3b82f6)', title: 'Cold coast', caption: 'Fishing village at dawn. The boats come back before the fog lifts.', likes: 152, comments: 11 },
  { author: 'sora', avatarColor: '#ec4899', gradient: 'linear-gradient(135deg,#ec4899,#f43f5e)', title: 'Festival flow', caption: 'Lanterns and music until 3am. Feet hurt, heart full.', likes: 788, comments: 102 },
  { author: 'dex', avatarColor: '#84cc16', gradient: 'linear-gradient(135deg,#84cc16,#22c55e)', title: 'Quiet field', caption: 'Bicycle, camera, nothing else. Found a field the map forgot.', likes: 63, comments: 4 },
  { author: 'zen', avatarColor: '#f43f5e', gradient: 'linear-gradient(135deg,#f43f5e,#a855f7)', title: 'Rooftop storm', caption: 'Storm rolled in fast. Stayed on the roof to watch it anyway.', likes: 925, comments: 141 },
];

export function generatePosts(offset, count) {
  const now = Date.now();
  const out = [];
  for (let i = 0; i < count; i++) {
    const base = BASE[(offset + i) % BASE.length];
    out.push({
      ...base,
      id: `p-${offset + i}`,
      timestamp: new Date(now - (offset + i) * 3600_000).toISOString(),
    });
  }
  return out;
}
