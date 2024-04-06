from scipy.stats import mannwhitneyu

def mwu_test_p(predhrl_final_cov, final_cov):
    stat, p = mannwhitneyu(predhrl_final_cov, final_cov, alternative="greater")
    return p