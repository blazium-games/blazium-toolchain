"use strict";

const fs = require("fs");
const path = require("path");

const version = process.argv[2] || "";
if (!/^\d+\.\d+\.\d+(?:[-+][0-9A-Za-z.-]+)?$/.test(version)) {
  console.error("usage: node npm/stamp-versions.cjs <semver>");
  process.exit(1);
}

const root = path.join(__dirname, "packages");
const stamped = [];
for (const name of fs.readdirSync(root)) {
  const pkgPath = path.join(root, name, "package.json");
  if (!fs.existsSync(pkgPath)) {
    continue;
  }
  const pkg = JSON.parse(fs.readFileSync(pkgPath, "utf8"));
  pkg.version = version;
  if (pkg.optionalDependencies) {
    for (const key of Object.keys(pkg.optionalDependencies)) {
      pkg.optionalDependencies[key] = version;
    }
  }
  fs.writeFileSync(pkgPath, JSON.stringify(pkg, null, 2) + "\n");
  const written = JSON.parse(fs.readFileSync(pkgPath, "utf8"));
  if (written.version !== version) {
    console.error(`${written.name} version is ${written.version}, want ${version}`);
    process.exit(1);
  }
  if (written.optionalDependencies) {
    for (const [key, depVersion] of Object.entries(written.optionalDependencies)) {
      if (depVersion !== version) {
        console.error(`${written.name} optional ${key} is ${depVersion}, want ${version}`);
        process.exit(1);
      }
    }
  }
  stamped.push(`${written.name}@${written.version}`);
}

if (stamped.length !== 5) {
  console.error(`expected 5 packages, stamped ${stamped.length}`);
  process.exit(1);
}
for (const line of stamped) {
  console.log(line);
}
