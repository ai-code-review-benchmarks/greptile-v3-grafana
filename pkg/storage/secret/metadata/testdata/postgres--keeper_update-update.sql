UPDATE
  "secret_keeper"
SET
  "guid" = 'abc',
  "name" = 'name',
  "namespace" = 'ns',
  "annotations" = '{"x":"XXXX"}',
  "labels" = '{"a":"AAA", "b", "BBBB"}',
  "created" = 1234,
  "created_by" = 'user:ryan',
  "updated" = 5678,
  "updated_by" = 'user:cameron',
  "description" = 'description',
  "type" = 'sql',
  "payload" = ''
WHERE 1 = 1 AND
  "namespace" = 'ns' AND
  "name" = 'name'
;
