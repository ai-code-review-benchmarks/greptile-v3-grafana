import { useState } from 'react';

import { Field, Input } from '@grafana/ui';

import { TextInputProps } from './TextInput';

export const NumberInputEditor = ({ onChange, currentOption }: TextInputProps) => {
  const [value, setValue] = useState(currentOption?.[0]);

  return (
    <Field label="Number input title">
      <Input
        defaultValue={value}
        onBlur={(event) => {
          setValue(event.currentTarget.value);
          onChange(event.currentTarget.value);
        }}
      />
    </Field>
  );
};
