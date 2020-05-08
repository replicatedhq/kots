import * as util from "util";
import * as _ from "lodash";
import * as fs from "fs";
import * as path from "path";
import { Schema } from "../schema";

exports.name = "generate";
exports.describe = "Generate SQL statements from fixtures";
exports.builder = {

};

exports.handler = async (argv) => {
  main(argv).catch((err) => {
    console.log(`Failed with error ${util.inspect(err)}`);
    process.exit(1);
  });
};

async function main(argv): Promise<any> {
  process.on('SIGTERM', function onSigterm () {
    process.exit();
  });

  console.log(`Converting all fixtures into SQL statements`);

  let allStatements: string[] = [];

  const files = fs.readdirSync("./fixtures");
  for (const file of files) {
    if (path.extname(file) === ".yaml") {
      console.log(`   begin converting ${file}`)

      const schema: Schema = new Schema();
      schema.parse(path.join("./fixtures", file));

      allStatements = allStatements.concat(schema.generateFixtures());

      console.log(`   finish converting ${file}`);
    }
  }

  fs.writeFileSync("./fixtures.sql", `/* Auto generated file. Do not edit by hand. */\n\n`);

  const schema = fs.readFileSync("./schema/fixtures.sql");
  fs.appendFileSync("./fixtures.sql", `${schema};\n`);

  for (const statement of allStatements) {
    fs.appendFileSync("./fixtures.sql", `${statement};\n`);
  }

  console.log(`All fixtures have been written as SQL statements`);

  process.exit(0);
}

