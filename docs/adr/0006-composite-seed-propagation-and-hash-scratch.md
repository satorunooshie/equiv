# 0006: Composite seed propagation

Child hashes are initialized from the parent `Hash.Seed()`, preventing nested
hashers from silently selecting unrelated random seeds.
