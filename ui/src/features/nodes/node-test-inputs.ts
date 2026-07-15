import type { TestInput } from "@/lib/api/endpoints"
import type { Outbound } from "@/lib/api/types"

export const nodeTestTypes = ["tcp", "http", "icmp"] as const
export type NodeTestType = typeof nodeTestTypes[number]

export function nodeTestInput(node: Outbound, type: NodeTestType): TestInput | null {
  if (!node.server || !node.port) return null
  return { tag: node.tag, test_type: type, server: node.server, port: node.port }
}

export function nodeTestInputs(nodes: Outbound[]) {
  return nodes.flatMap((node) => nodeTestTypes.flatMap((type) => {
    const input = nodeTestInput(node, type)
    return input ? [input] : []
  }))
}
