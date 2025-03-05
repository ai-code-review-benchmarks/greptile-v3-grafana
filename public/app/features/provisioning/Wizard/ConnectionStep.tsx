import { Controller, useFormContext } from 'react-hook-form';

import { Field, Input, Combobox, Stack, FieldSet } from '@grafana/ui';

import { WizardFormData } from './types';

const typeOptions = [
  { label: 'GitHub', value: 'github' },
  { label: 'Local', value: 'local' },
];

// const modeOptions = [
//   {
//     value: 'instance',
//     label: 'Connect your Grafana instance to an empty repository',
//     description:
//       'Export all dashboards from this instance to a new, empty repository. After setup, all dashboards in this instance will be saved and managed exclusively through this repository.',
//   },
//   {
//     value: 'import',
//     label: 'Import dashboards from an existing repository',
//     description:
//       'Use dashboards from your GitHub repository to populate an empty Grafana instance. After setup, all dashboards in the repository will be automatically provisioned into this instance.',
//   },
//   {
//     value: 'folder',
//     label: 'Connect a specific folder to your repository',
//     description:
//       'Save and manage dashboards from a selected folder while keeping others separate. You can create multiple connections between different folders and repositories.',
//   },
// ];

export function ConnectionStep() {
  const {
    register,
    control,
    setValue,
    watch,
    formState: { errors },
  } = useFormContext<WizardFormData>();

  const workflows = watch('repository.workflows');

  return (
    <FieldSet label="1. Set up your repository connection details">
      <Stack direction="column" gap={2}>
        <Field
          label="Repository type"
          description="Choose the type of repository where your dashboards will be stored."
          required
        >
          <Controller
            name="repository.type"
            control={control}
            rules={{ required: true }}
            render={({ field: { ref, onChange, ...field } }) => (
              <Combobox
                {...field}
                options={typeOptions}
                placeholder="Select repository type"
                onChange={(value) => {
                  onChange(value?.value);
                  setValue(
                    'repository.workflows',
                    workflows.filter((workflow) => workflow !== 'branch')
                  );
                }}
              />
            )}
          />
        </Field>
        <Field
          label="Display name"
          description="Add a clear name for this configuration"
          error={errors.repository?.title?.message}
          invalid={!!errors.repository?.title}
        >
          <Input
            {...register('repository.title', { required: 'This field is required.' })}
            placeholder="My repository connection"
          />
        </Field>

        {/*<Stack direction="column" gap={2}>*/}
        {/*  <Controller*/}
        {/*    name="repository.sync.target"*/}
        {/*    control={control}*/}
        {/*    render={({ field: { onChange, value } }) => (*/}
        {/*      <>*/}
        {/*        {modeOptions.map((option) => (*/}
        {/*          <Card key={option.value} isSelected={value === option.value} onClick={() => onChange(option.value)}>*/}
        {/*            <Card.Heading>{option.label}</Card.Heading>*/}
        {/*            <Card.Description>{option.description}</Card.Description>*/}
        {/*          </Card>*/}
        {/*        ))}*/}
        {/*      </>*/}
        {/*    )}*/}
        {/*  />*/}
        {/*</Stack>*/}
      </Stack>
    </FieldSet>
  );
}
