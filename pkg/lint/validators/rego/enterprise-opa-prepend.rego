package kots.enterprise

# Files set with the contents of each file as json
files[output] {
  file := input[_]
  output := {
    "name": file.name,
    "path": file.path,
    "content": yaml.unmarshal(file.content),
    "docIndex": object.get(file, "docIndex", 0),
    "allowDuplicates": object.get(file, "allowDuplicates", false)
  }
}

# Returns the string value of x
string(x) = y {
	y := split(yaml.marshal(x), "\n")[0]
}

# A set containing ALL the specs for each file
# 3 levels deep. "specs" rule for each level
specs[output] {
  file := files[_]
  spec := file.content.spec # 1st level
  output := {
    "path": file.path,
    "spec": spec,
    "field": "spec",
    "docIndex": file.docIndex
  }
}
specs[output] {
  file := files[_]
  spec := file.content[key].spec # 2nd level
  field := concat(".", [string(key), "spec"])
  output := {
    "path": file.path,
    "spec": spec,
    "field": field,
    "docIndex": file.docIndex
  }
}
specs[output] {
  file := files[_]
  spec := file.content[key1][key2].spec # 3rd level
  field := concat(".", [string(key1), string(key2), "spec"])
  output := {
    "path": file.path,
    "spec": spec,
    "field": field,
    "docIndex": file.docIndex
  }
}