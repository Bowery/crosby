# Bowery Compiler Cache
A wrapper around gcc that will look get the result of a gcc call

## TODO
- Change sources schema from map[string] interface{} to:
  ```
    type Source struct {
      Results: []bson.ObjectId, // Ids of the resulting files
      Files: map[string] string, // relative file paths and resulting md5 sums
      Arch: string, // the operating system/architecture combination
      Args: string, // strings.Join(flag.Args(), " ")
    }
  ```
- Support more than just gcc. This would make npm install a hell of a lot faster
- Better Logging
- Detect false alarms (files being added). Right now if a source has some of the files with the same md5 sum then it will likely grab the wrong project form the cache.
