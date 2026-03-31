import { $ } from "bun";

// export {};

// 清理并创建 dist 目录
console.log("Cleaning up ...");
await $`rm -rf ./dist && mkdir -p ./dist`;

console.log("Building ...");
// 构建项目
const result = await Bun.build({
  entrypoints: ["./src/app.ts"],
  outdir: "./dist",
  // 其他配置...
  splitting: true,
  minify: true,
  watch: process.env.WATCH === "true",
});

if (result.success) {
  console.log("Build successful!");

  // 在构建成功后手动拷贝目录
  console.log("Copying public assets...");
  await $`cp ./logo.svg ./favicon.svg ./dist/`;
  // await $`cp ./src/style/highlight.css ./src/style/app.css ./logo.svg ./favicon.svg ./dist/`;
} else {
  console.error("Build failed!");
}

await $`ls -lh ./dist`;