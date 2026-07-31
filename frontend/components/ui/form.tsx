"use client"

import * as React from "react"
import { Slot } from "radix-ui"
import {
  Controller,
  FormProvider,
  useFormContext,
  useFormState,
  type ControllerProps,
  type FieldPath,
  type FieldValues,
} from "react-hook-form"

import {
  Field,
  FieldDescription,
  FieldError,
  FieldLabel,
} from "@/components/ui/field"

// The radix-nova registry has retired shadcn's `form` component in favour of
// `field`, which knows nothing about react-hook-form. This is the bridge: the
// familiar Form* API on top of the Field* primitives, so a field declares its
// label, control, description and message and the id / aria-describedby /
// aria-invalid wiring follows automatically.

const Form = FormProvider

type FormFieldContextValue<
  TFieldValues extends FieldValues = FieldValues,
  TName extends FieldPath<TFieldValues> = FieldPath<TFieldValues>,
> = {
  name: TName
}

const FormFieldContext = React.createContext<FormFieldContextValue | null>(null)

function FormField<
  TFieldValues extends FieldValues = FieldValues,
  TName extends FieldPath<TFieldValues> = FieldPath<TFieldValues>,
>({ ...props }: ControllerProps<TFieldValues, TName>) {
  return (
    <FormFieldContext.Provider value={{ name: props.name }}>
      <Controller {...props} />
    </FormFieldContext.Provider>
  )
}

const FormItemContext = React.createContext<{ id: string } | null>(null)

function useFormField() {
  const fieldContext = React.useContext(FormFieldContext)
  const itemContext = React.useContext(FormItemContext)
  const form = useFormContext()
  const formState = useFormState({ name: fieldContext?.name as string })

  if (!fieldContext) {
    throw new Error("useFormField must be used inside a <FormField>")
  }
  if (!itemContext) {
    throw new Error("useFormField must be used inside a <FormItem>")
  }

  const { id } = itemContext

  return {
    id,
    name: fieldContext.name,
    formItemId: `${id}-form-item`,
    formDescriptionId: `${id}-form-item-description`,
    formMessageId: `${id}-form-item-message`,
    ...form.getFieldState(fieldContext.name, formState),
  }
}

function FormItem({ ...props }: React.ComponentProps<typeof Field>) {
  const id = React.useId()

  return (
    <FormItemContext.Provider value={{ id }}>
      <FormItemField {...props} />
    </FormItemContext.Provider>
  )
}

// Split out because `Field` needs the error state that only becomes readable
// once FormItemContext is in scope — a component cannot consume the context it
// is itself providing.
function FormItemField({ ...props }: React.ComponentProps<typeof Field>) {
  const { error } = useFormField()

  return <Field data-invalid={error ? true : undefined} {...props} />
}

function FormLabel({ ...props }: React.ComponentProps<typeof FieldLabel>) {
  const { formItemId } = useFormField()

  return <FieldLabel htmlFor={formItemId} {...props} />
}

function FormControl({ ...props }: React.ComponentProps<typeof Slot.Root>) {
  const { error, formItemId, formDescriptionId, formMessageId } = useFormField()

  return (
    <Slot.Root
      id={formItemId}
      aria-describedby={
        error ? `${formDescriptionId} ${formMessageId}` : formDescriptionId
      }
      aria-invalid={error ? true : undefined}
      {...props}
    />
  )
}

function FormDescription({
  ...props
}: React.ComponentProps<typeof FieldDescription>) {
  const { formDescriptionId } = useFormField()

  return <FieldDescription id={formDescriptionId} {...props} />
}

function FormMessage({ ...props }: React.ComponentProps<typeof FieldError>) {
  const { error, formMessageId } = useFormField()

  return <FieldError id={formMessageId} errors={error ? [error] : []} {...props} />
}

export {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
  useFormField,
}
