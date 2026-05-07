#ifndef NCNN_BRIDGE_H
#define NCNN_BRIDGE_H

#ifdef __cplusplus
extern "C" {
#endif

/*
 * NcnnEmbedder wraps an ncnn net loaded from .param + .bin files.
 * Create once at startup; reuse across Embed calls.
 */
typedef struct NcnnEmbedder NcnnEmbedder;

/* Load the ncnn model from param_path and bin_path. Returns NULL on error. */
NcnnEmbedder* ncnn_embedder_create(const char* param_path, const char* bin_path);

/* Free all resources. */
void ncnn_embedder_destroy(NcnnEmbedder* e);

/*
 * Run one forward pass.
 *
 * input_ids, attn_mask, token_type_ids: int32 arrays of length seq_len
 * output: caller-allocated float32 array of length embed_dim (384)
 *
 * Returns 0 on success, non-zero on error.
 *
 * Note: output is the last_hidden_state tensor flattened to
 * [seq_len * embed_dim]; mean pooling is performed in Go.
 *
 * Blob names are those produced by PNNX for bert_minilm.pt with
 * inputname="input_ids,attention_mask,token_type_ids" and
 * outputname="last_hidden_state".  Verify against the actual converted
 * model's .param file if inference fails.
 */
int ncnn_embedder_run(NcnnEmbedder* e,
                      const int* input_ids,
                      const int* attn_mask,
                      const int* token_type_ids,
                      int seq_len,
                      float* output,
                      int embed_dim);

#ifdef __cplusplus
}
#endif

#endif /* NCNN_BRIDGE_H */
