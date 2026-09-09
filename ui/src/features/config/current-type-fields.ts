import type { PolicyFieldSpec } from "@/features/policy/policy-form-model"

/** 先选择协议，再清理条件字段，避免其他协议的同名字段被误删。 */
export function currentTypeFields(fields: readonly PolicyFieldSpec[], type: string): PolicyFieldSpec[] {
  return fields.filter((field) => {
    const conditions = field.when ? Array.isArray(field.when) ? field.when : [field.when] : []
    return conditions.every((condition) => {
      if (condition.path !== "type") return true
      const allowed = Array.isArray(condition.is) ? condition.is : [condition.is]
      return allowed.includes(type)
    })
  })
}
