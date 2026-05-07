//go:build ncnn

#include "ncnn_bridge.h"
#include "ncnn/c_api.h"

#include <stdlib.h>
#include <string.h>
#include <stdio.h>

struct NcnnEmbedder {
    ncnn_net_t net;
};

NcnnEmbedder* ncnn_embedder_create(const char* param_path, const char* bin_path) {
    NcnnEmbedder* e = (NcnnEmbedder*)calloc(1, sizeof(NcnnEmbedder));
    if (!e) return NULL;

    e->net = ncnn_net_create();
    if (!e->net) {
        free(e);
        return NULL;
    }

    /* Single-threaded: embedding runs in a background goroutine; avoid
     * contention with the Go HTTP server's scheduler. */
    ncnn_option_t opt = ncnn_option_create();
    ncnn_option_set_num_threads(opt, 1);
    ncnn_option_set_use_vulkan_compute(opt, 0);
    ncnn_net_set_option(e->net, opt);
    ncnn_option_destroy(opt);

    if (ncnn_net_load_param(e->net, param_path) != 0) {
        fprintf(stderr, "ncnn_bridge: failed to load param: %s\n", param_path);
        ncnn_net_destroy(e->net);
        free(e);
        return NULL;
    }

    if (ncnn_net_load_model(e->net, bin_path) != 0) {
        fprintf(stderr, "ncnn_bridge: failed to load model: %s\n", bin_path);
        ncnn_net_destroy(e->net);
        free(e);
        return NULL;
    }

    return e;
}

void ncnn_embedder_destroy(NcnnEmbedder* e) {
    if (!e) return;
    if (e->net) ncnn_net_destroy(e->net);
    free(e);
}

int ncnn_embedder_run(NcnnEmbedder* e,
                      const int* input_ids,
                      const int* attn_mask,
                      const int* token_type_ids,
                      int seq_len,
                      float* output,
                      int embed_dim) {
    if (!e || !e->net) return -1;

    ncnn_extractor_t ex = ncnn_extractor_create(e->net);
    if (!ex) return -2;

    /* Create 1D float mats and copy integer data into them.
     * ncnn's Embed (embedding lookup) layer reads the raw bytes as int32;
     * a float32 mat has the same element size (4 bytes), so this is correct.
     * If the PNNX conversion produced a model expecting a different type,
     * verify blob names in the .param file and adjust accordingly. */
    ncnn_mat_t mat_ids   = ncnn_mat_create_1d(seq_len, NULL);
    ncnn_mat_t mat_mask  = ncnn_mat_create_1d(seq_len, NULL);
    ncnn_mat_t mat_types = ncnn_mat_create_1d(seq_len, NULL);

    if (!mat_ids || !mat_mask || !mat_types) {
        ncnn_mat_destroy(mat_ids);
        ncnn_mat_destroy(mat_mask);
        ncnn_mat_destroy(mat_types);
        ncnn_extractor_destroy(ex);
        return -3;
    }

    memcpy(ncnn_mat_get_data(mat_ids),   input_ids,      seq_len * sizeof(int));
    memcpy(ncnn_mat_get_data(mat_mask),  attn_mask,      seq_len * sizeof(int));
    memcpy(ncnn_mat_get_data(mat_types), token_type_ids, seq_len * sizeof(int));

    /* Input blob names match PNNX export with inputname=
     * "input_ids,attention_mask,token_type_ids". */
    ncnn_extractor_input(ex, "input_ids",      mat_ids);
    ncnn_extractor_input(ex, "attention_mask", mat_mask);
    ncnn_extractor_input(ex, "token_type_ids", mat_types);

    ncnn_mat_destroy(mat_ids);
    ncnn_mat_destroy(mat_mask);
    ncnn_mat_destroy(mat_types);

    /* Extract last_hidden_state: shape [seq_len, embed_dim] in ncnn (HWC). */
    ncnn_mat_t out = NULL;
    int ret = ncnn_extractor_extract(ex, "last_hidden_state", &out);
    ncnn_extractor_destroy(ex);

    if (ret != 0 || !out) {
        ncnn_mat_destroy(out);
        return -4;
    }

    /* Copy output: ncnn mat is [w=embed_dim, h=seq_len] or [w=seq_len*embed_dim].
     * Copy all floats; Go-side mean pooling uses seq_len to slice correctly. */
    int total = ncnn_mat_get_w(out) * ncnn_mat_get_h(out);
    if (total > seq_len * embed_dim) total = seq_len * embed_dim;
    memcpy(output, ncnn_mat_get_data(out), total * sizeof(float));

    ncnn_mat_destroy(out);
    return 0;
}
