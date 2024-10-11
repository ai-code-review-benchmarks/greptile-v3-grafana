import { action } from '@storybook/addon-actions';
import { Meta, StoryFn, StoryObj } from '@storybook/react';
import { Chance } from 'chance';
import React, { ComponentProps, useCallback, useEffect, useState } from 'react';

import { Alert } from '../Alert/Alert';
import { Field } from '../Forms/Field';

import { Combobox, ComboboxOption } from './Combobox';

const chance = new Chance();

type PropsAndCustomArgs = ComponentProps<typeof Combobox> & { numberOfOptions: number };

const meta: Meta<PropsAndCustomArgs> = {
  title: 'Forms/Combobox',
  component: Combobox,
  args: {
    loading: undefined,
    invalid: undefined,
    width: undefined,
    placeholder: 'Select an option...',
    options: [
      { label: 'Apple', value: 'apple' },
      { label: 'Banana', value: 'banana' },
      { label: 'Carrot', value: 'carrot' },
      // Long label to test overflow
      {
        label:
          'Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua.',
        value: 'long-text',
      },
      { label: 'Dill', value: 'dill' },
      { label: 'Eggplant', value: 'eggplant' },
      { label: 'Fennel', value: 'fennel' },
      { label: 'Grape', value: 'grape' },
      { label: 'Honeydew', value: 'honeydew' },
      { label: 'Iceberg Lettuce', value: 'iceberg-lettuce' },
      { label: 'Jackfruit', value: 'jackfruit' },
      { label: '1', value: 1 },
      { label: '2', value: 2 },
      { label: '3', value: 3 },
    ],
    value: 'banana',
  },

  render: (args) => <BasicWithState {...args} />,
  decorators: [InDevDecorator],
};

const BasicWithState: StoryFn<typeof Combobox> = (args) => {
  const [value, setValue] = useState(args.value);
  return (
    <Field label="Test input" description="Input with a few options">
      <Combobox
        id="test-combobox"
        {...args}
        value={value}
        onChange={(val) => {
          setValue(val?.value || null);
          action('onChange')(val);
        }}
      />
    </Field>
  );
};

type Story = StoryObj<typeof Combobox>;

export const Basic: Story = {};

async function generateOptions(amount: number): Promise<ComboboxOption[]> {
  return Array.from({ length: amount }, (_, index) => ({
    label: chance.sentence({ words: index % 5 }),
    value: chance.guid(),
  }));
}

const ManyOptionsStory: StoryFn<PropsAndCustomArgs> = ({ numberOfOptions, ...args }) => {
  const [value, setValue] = useState<string | null>(null);
  const [options, setOptions] = useState<ComboboxOption[]>([]);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    setTimeout(() => {
      generateOptions(numberOfOptions).then((options) => {
        setIsLoading(false);
        setOptions(options);
        setValue(options[5].value);
        console.log("I've set stuff");
      });
    }, 1000);
  }, [numberOfOptions]);

  return (
    <Combobox
      {...args}
      loading={isLoading}
      options={options}
      value={value}
      onChange={(opt) => {
        setValue(opt?.value || null);
        action('onChange')(opt);
      }}
    />
  );
};

export const AutoSize: StoryObj<PropsAndCustomArgs> = {
  args: {
    width: 'auto',
    minWidth: 5,
    maxWidth: 200,
  },
};

export const ManyOptions: StoryObj<PropsAndCustomArgs> = {
  args: {
    numberOfOptions: 1e5,
    options: undefined,
    value: undefined,
  },
  render: ManyOptionsStory,
};

export const CustomValue: StoryObj<PropsAndCustomArgs> = {
  args: {
    createCustomValue: true,
  },
};

const AsyncStory: StoryFn<PropsAndCustomArgs> = (args) => {
  const [value, setValue] = useState<string | number | null>(null);

  // This simulates a kind of search API call
  const loadOptions = useCallback(
    (inputValue: string) => {
      return fakeSearchAPI(`http://example.com/search?query=${inputValue}`);
    },
    [args.options]
  );

  return (
    <Field label="Test input" description="Input with a few options">
      <Combobox
        id="test-combobox"
        placeholder="Select an option"
        options={loadOptions}
        value={value}
        onChange={(val) => {
          action('onChange')(val);
          setValue(val?.value || null);
        }}
        createCustomValue={args.createCustomValue}
      />
    </Field>
  );
};

export const Async: StoryObj<PropsAndCustomArgs> = {
  render: AsyncStory,
};

export default meta;

function InDevDecorator(Story: React.ElementType) {
  return (
    <div>
      <Alert title="This component is still in development!" severity="info">
        Combobox is still in development and not able to be used externally.
        <br />
        Within the Grafana repo, it can be used by importing it from{' '}
        <span style={{ fontFamily: 'monospace' }}>@grafana/ui/src/unstable</span>
      </Alert>
      <Story />
    </div>
  );
}

let fakeApiOptions: Array<ComboboxOption<string | number>>;
async function fakeSearchAPI(urlString: string) {
  const searchParams = new URL(urlString).searchParams;

  if (!fakeApiOptions) {
    fakeApiOptions = await generateOptions(1000);
  }

  const searchQuery = searchParams.get('query')?.toLowerCase();

  if (!searchQuery || searchQuery.length === 0) {
    return Promise.resolve(fakeApiOptions.slice(0, 100));
  }

  const filteredOptions = Promise.resolve(
    fakeApiOptions.filter((opt) => opt.label?.toLowerCase().includes(searchQuery))
  );

  const delay = searchQuery.length % 2 === 0 ? 200 : 1000;
  return new Promise<Array<ComboboxOption<string | number>>>((resolve) => {
    setTimeout(() => resolve(filteredOptions), delay);
  });
}
