const fs = require('fs');
const path = require('path');
const DIR = path.join(__dirname, '..', 'tokens');

module.exports = {
  save(name, data) {
    fs.writeFileSync(path.join(DIR, `${name}.json`), JSON.stringify(data, null, 2));
  },
  load(name) {
    const file = path.join(DIR, `${name}.json`);
    if (!fs.existsSync(file)) return null;
    try { return JSON.parse(fs.readFileSync(file, 'utf8')); } catch { return null; }
  },
  clear(name) {
    const file = path.join(DIR, `${name}.json`);
    if (fs.existsSync(file)) fs.unlinkSync(file);
  },
};
