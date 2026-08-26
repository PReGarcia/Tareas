const fs = require("fs");
const path = require("path");
const { execFileSync } = require("child_process");

const root = __dirname;
const dist = path.join(root, "dist");

function cp(src, dest) {
  fs.mkdirSync(path.dirname(dest), { recursive: true });
  fs.copyFileSync(src, dest);
}

// 1) Limpiar dist
fs.rmSync(dist, { recursive: true, force: true });
fs.mkdirSync(path.join(dist, "assets"), { recursive: true });
fs.mkdirSync(path.join(dist, "fonts"), { recursive: true });

// 2) Tailwind CSS -> dist/assets/app.css (minificado)
const tw = path.join(root, "node_modules", ".bin", "tailwindcss");
execFileSync(
  tw,
  ["-i", path.join(root, "src", "input.css"), "-o", path.join(dist, "assets", "app.css"), "--minify"],
  { stdio: "inherit" }
);

// 3) JS de la app
cp(path.join(root, "src", "app.js"), path.join(dist, "assets", "app.js"));

// 3b) Librerías vendor (offline / self-hosted)
const vendor = {
  "htmx.org/dist/htmx.min.js": "assets/vendor/htmx.min.js",
  "alpinejs/dist/cdn.min.js": "assets/vendor/alpine.min.js",
  "sortablejs/Sortable.min.js": "assets/vendor/sortable.min.js",
};
for (const [srcRel, destRel] of Object.entries(vendor)) {
  const src = path.join(root, "node_modules", ...srcRel.split("/"));
  if (fs.existsSync(src)) cp(src, path.join(dist, destRel));
  else console.warn("vendor no encontrado:", srcRel);
}

// 4) index.html
cp(path.join(root, "src", "index.html"), path.join(dist, "index.html"));

// 5) Fuentes Inter (self-hosted)
const fontDir = path.join(root, "node_modules", "@fontsource", "inter", "files");
const weights = ["400", "500", "600", "700"];
for (const w of weights) {
  const name = `inter-latin-${w}-normal.woff2`;
  const src = path.join(fontDir, name);
  if (fs.existsSync(src)) cp(src, path.join(dist, "fonts", name));
}

console.log("build ok ->", dist);
