SELECT
  `guid`,
  `name`,
  `namespace`,
  `annotations`,
  `labels`,
  `created`,
  `created_by`,
  `updated`,
  `updated_by`,
  `title`,
  `type`,
  `payload`
FROM
  `secret_keeper`
WHERE 1 = 1 AND
  `namespace` = 'ns' AND
  `name` = 'name'
ORDER BY `updated` DESC
FOR UPDATE
;
