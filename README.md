# bowery gcc wrapper
gives you the result of a gcc call without running gcc if someone has compiled that same code before.

## TODO
- Better Logging
- Detect false alarms (files being added). Right now if a source has some of the files with the same md5 sum then it will likely grab the wrong project form the cache.
