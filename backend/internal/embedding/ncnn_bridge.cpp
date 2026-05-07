//go:build ncnn

#include "ncnn_bridge.h"
#include <ncnn/net.h>
#include <ncnn/layer.h>
#include <string.h>
#include <stdio.h>
#include <stdlib.h>

// pnnx emits Tensor.to for dtype casts (e.g. float, bool) but strips the
// dtype param as "ignored", so the layer is always a pure passthrough.
class TensorToLayer : public ncnn::Layer {
public:
    TensorToLayer() { one_blob_only = true; }

    int forward(const ncnn::Mat& bottom_blob, ncnn::Mat& top_blob,
                const ncnn::Option& /*opt*/) const override {
        top_blob = bottom_blob;
        return 0;
    }
};
DEFINE_LAYER_CREATOR(TensorToLayer)

// pnnx emits Tensor.masked_fill for BERT's attention-mask construction:
//   (1 - attention_mask).masked_fill(bool_mask, fill_value)
// It strips the fill_value param, so we default to -10000.0 — the standard
// additive attention-mask sentinel used by HuggingFace BERT.
class TensorMaskedFillLayer : public ncnn::Layer {
public:
    TensorMaskedFillLayer() : value(-10000.0f) { one_blob_only = false; }

    int load_param(const ncnn::ParamDict& pd) override {
        value = pd.get(0, -10000.0f);
        return 0;
    }

    int forward(const std::vector<ncnn::Mat>& bottom_blobs,
                std::vector<ncnn::Mat>& top_blobs,
                const ncnn::Option& /*opt*/) const override {
        const ncnn::Mat& data = bottom_blobs[0];
        const ncnn::Mat& mask = bottom_blobs[1];
        ncnn::Mat& out = top_blobs[0];

        out = data.clone();

        int total = out.total();
        float* ptr  = out;
        const float* mptr = mask;
        for (int i = 0; i < total; i++) {
            if (mptr[i] != 0.f) ptr[i] = value;
        }
        return 0;
    }

    float value;
};
DEFINE_LAYER_CREATOR(TensorMaskedFillLayer)

struct NcnnEmbedder {
    ncnn::Net* net;
};

extern "C" {

NcnnEmbedder* ncnn_embedder_create(const char* param_path, const char* bin_path) {
    NcnnEmbedder* e = (NcnnEmbedder*)calloc(1, sizeof(NcnnEmbedder));
    if (!e) return nullptr;

    e->net = new ncnn::Net();
    e->net->register_custom_layer("Tensor.to",          TensorToLayer_layer_creator);
    e->net->register_custom_layer("Tensor.masked_fill", TensorMaskedFillLayer_layer_creator);
    e->net->opt.num_threads = 1;
    e->net->opt.use_vulkan_compute = false;

    if (e->net->load_param(param_path) != 0) {
        fprintf(stderr, "ncnn_bridge: failed to load param: %s\n", param_path);
        delete e->net;
        free(e);
        return nullptr;
    }

    if (e->net->load_model(bin_path) != 0) {
        fprintf(stderr, "ncnn_bridge: failed to load model: %s\n", bin_path);
        delete e->net;
        free(e);
        return nullptr;
    }

    return e;
}

void ncnn_embedder_destroy(NcnnEmbedder* e) {
    if (!e) return;
    delete e->net;
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

    ncnn::Extractor ex = e->net->create_extractor();

    // ncnn Embed layers read raw bytes as int32; Mat stores float32 with the
    // same 4-byte element size, so memcpy is the correct way to pass indices.
    ncnn::Mat mat_ids(seq_len);
    ncnn::Mat mat_mask(seq_len);
    ncnn::Mat mat_types(seq_len);

    memcpy(mat_ids.data,   input_ids,      seq_len * sizeof(int));
    memcpy(mat_mask.data,  attn_mask,      seq_len * sizeof(int));
    memcpy(mat_types.data, token_type_ids, seq_len * sizeof(int));

    // ncnn blob names are auto-assigned by pnnx (in0/in1/in2/out0).
    // inputname/outputname in the pnnx command only rename pnnx-level blobs.
    ex.input("in0", mat_ids);    // input_ids
    ex.input("in1", mat_mask);   // attention_mask
    ex.input("in2", mat_types);  // token_type_ids

    ncnn::Mat out;
    int ret = ex.extract("out0", out);
    if (ret != 0) return -4;

    int total = out.w * out.h * out.c;
    if (total > seq_len * embed_dim) total = seq_len * embed_dim;
    memcpy(output, out.data, total * sizeof(float));

    return 0;
}

} // extern "C"
