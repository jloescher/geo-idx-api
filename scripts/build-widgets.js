import { build } from "vite";
import fs from "node:fs/promises";
import path from "node:path";
import JavaScriptObfuscator from "javascript-obfuscator";

const root = process.cwd();
const devOutDir = path.join(root, "dist/widgets");
const prodOutDir = path.join(root, "public/js/widgets/prod");

await build({
  configFile: false,
  build: {
    minify: false,
    sourcemap: true,
    outDir: devOutDir,
    emptyOutDir: true,
    lib: {
      entry: path.join(root, "resources/js/widgets/loader.js"),
      formats: ["iife"],
      name: "QuantyraGeoIDXLoader",
      fileName: () => "loader.js",
    },
  },
});

const loaderSource = await fs.readFile(path.join(devOutDir, "loader.js"), "utf8");
const obfuscated = JavaScriptObfuscator.obfuscate(loaderSource, {
  compact: true,
  controlFlowFlattening: true,
  deadCodeInjection: true,
  identifierNamesGenerator: "hexadecimal",
  rotateStringArray: true,
  stringArray: true,
  stringArrayEncoding: ["base64"],
  stringArrayShuffle: true,
  transformObjectKeys: true,
}).getObfuscatedCode();

await fs.mkdir(prodOutDir, { recursive: true });
await fs.writeFile(path.join(prodOutDir, "loader.js"), obfuscated, "utf8");
